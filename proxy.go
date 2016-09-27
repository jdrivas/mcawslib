package mclib

import(
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/route53"
  "github.com/Sirupsen/logrus"

  // "awslib"
  "github.com/jdrivas/awslib"
)


// TODO: Is there state that we need to save on the 
// Proxy? Probably as we addd plugins.
type Proxy struct {
  Name string
  ClusterName string
  // TODO: Let's wait on adding a point to the hub server directly.
  // You can find it if you need it.
  // Hub *Server

  PublicProxyIp string
  PrivateProxyIp string
  ProxyPort Port
  RconPort Port
  RconPassword string
  Rcon *Rcon

  TaskArn string
  AWSSession *session.Session
}

// These are critical constants to the behavior of
// the system, specifically ecs-craft and the task-definitions
// all rely on using these names correctly.
const (
  DefaultProxyTaskDefinition = BungeeProxyRandomPortTaskDef
  BungeeProxyServerContainerName = "bungee"
  BungeeProxyHubServerContainerName = "minecraft-hub"
  BungeeProxyHubControllerContainerName = "minecraft-control"
)
// These task-definitions are built from the corresponding
// task definitions in go/src/mclib/task-definitions.
// They use the variables defined above. 
// There is currently no auto-generation of these configs
// and so they are kept in sync by hand!
const(  
  BungeeProxyDefaultPortTaskDef = "bungee-default"
  BungeeProxyRandomPortTaskDef = "bungee-random"
)

func NewProxy(name, clusterName, publicIp, privateIp, taskArn, rconPw string,
  proxyPort, rconPort int64, sess *session.Session ) (p *Proxy) {

  p = new(Proxy)
  p.Name = name
  p.ClusterName = clusterName
  p.PublicProxyIp = publicIp
  p.PrivateProxyIp = privateIp
  p.ProxyPort = Port(proxyPort)
  p.RconPort = Port(rconPort)
  p.RconPassword = rconPw
  p.TaskArn = taskArn
  p.AWSSession = sess

  return p
}

// This is a convenience for getting basic proxy information as well as all the data associated
// with a task running a Proxy. Many times you can choose to ignore the dtm returned. But equally,
// you may want all that data, rather than getting all the proxies then doing another DTM call
// since we need that information to get the list of proxies, we return it here.
func GetProxies(clusterName string, sess *session.Session) (p []*Proxy, dtm awslib.DeepTaskMap, err error) {

  p = make([]*Proxy, 0)
  dtm, err = awslib.GetDeepTasks(clusterName, sess)
  if err != nil { return p, dtm, err }

  for arn, dt := range dtm {
    proxy, ok := GetProxyFromTask(dt, arn, sess)
    if ok {
      p = append(p, proxy)
    }
  }

  return p, dtm, err
}

// Checks all of the conatiners for one who has an env with
// env[RoleKey] == CraftProxyRole.
// TODO: There are certainly faster ways of doing this, but for the moment.
// this seems like the most robust in the face of change and given the tools
// avaialble. In particular it would be better if I could meta-tag containers
// and tasks, but AWS seems to only keep the Docker Labels on the TaskDefinition.
// Or I could just look for the Environment for BungeeProxyServerContainerName.
// If it's there, then I'm done ..... hnmmmm.
// func isProxy(dt *awslib.DeepTask) (bool) {
//   _, ok := dt.GetEnvironment(BungeeProxyServerContainerName)
//   return ok
// }

func GetProxy(clusterName, taskArn string, sess *session.Session) (p *Proxy, err error) {
  dt, err := awslib.GetDeepTask(clusterName, taskArn, sess)
  var ok bool
  if err == nil {
    p, ok = GetProxyFromTask(dt, taskArn, sess)
    if !ok { err = fmt.Errorf("This task does not appear to be a proxy: %s", taskArn) }
  }
  return p, err
}

func GetProxyFromName(proxyName, clusterName string, sess *session.Session) (p *Proxy, err error) {
  proxies, _, err := GetProxies(clusterName, sess)
  if err == nil {
    for _, proxy := range proxies {
      if proxy.Name == proxyName {
        p = proxy
        break
      }
    }
  }
  return p, err
}

// TODO: THIS IS IMPORTANT. We need to check the DNS to see if we're 
// currently attached to tne network or not.  Suggested updates 
// include: add a new field to the Proxy struct which is the
// actual DNS address for this proxy and have this function AND 
// ONLY this function fill it out.
//
// TODO: THIS IS IMPORTANT. We are currently equating wether a task is a proxy task
// by virtue of it having a container with the proxy name. This may not be the best
// thing. On the other hand I haven't got anything better yet.
// We are using the ROLE environment variable that we might want to check as well .....
func GetProxyFromTask(dt *awslib.DeepTask, taskArn string, sess *session.Session) (p *Proxy, ok bool) {
  proxyEnv, ok := dt.GetEnvironment(BungeeProxyServerContainerName)
  if ok {
    proxyPort := int64(0)
    if port, ok := dt.PortHostBinding(BungeeProxyServerContainerName, ProxyPortDefault); ok {
      proxyPort = port
    }
    rconPort := int64(0)
    if port, ok := dt.PortHostBinding(BungeeProxyServerContainerName, RconPortDefault); ok {
      rconPort = port
    }

    p = NewProxy(
      proxyEnv[ServerNameKey], dt.ClusterName(), dt.PublicIpAddress(), dt.PrivateIpAddress(), taskArn,
      proxyEnv[RconPasswordKey], proxyPort, rconPort, sess)
  }

  return p, ok
}



func (p *Proxy) PublicIpAddress() (string) {
  return fmt.Sprintf("%s:%d", p.PublicProxyIp, p.ProxyPort)
}

func (p *Proxy) RconAddress() (string) {
  return fmt.Sprintf("%s:%d", p.PrivateProxyIp, p.RconPort)
}

// TODO: This should do a DNS lookup to make sure ....
func (p *Proxy) PublicDNSName() (string) {
  return p.GetDomainName()
}

func (p *Proxy) GetDomainName() (dn string) {
  // dn = p.Name + ".hood.momentlabs.io"
  dn = p.Name + ".hoods.momentlabs.io"
  return dn
}

// This can be a little expensive. It makes 4 calls to AWS.
func (p *Proxy) GetDeepTask() (dt *awslib.DeepTask, err error) {
  return awslib.GetDeepTask(p.ClusterName, p.TaskArn, p.AWSSession)
}

// Associate with a possibly new ElasticIP address and publish 
// (or reuse) a DNS entry for this proxy.
// type AttachNetworkResp struct {
//   EIPStatus *ec2.AllocateAddressOutput // AllocationId *string, Domain *string, PublicIp *string
//   DNSStatus *route53.ChangeInfo // Comment *string, Id *string, Status *string, SubmittedAt *time.Time
//   AssocId *string // association id for the EIP to Instance association.
//   DNSAddress string
// }

// This should probably be longer ......
const DefaultProxyTTL int64 = 60

// Grab an IP then push it to DNS and attach to the instance.
// Actually I don't really need the EIP .... Just an attach.
func (p *Proxy) AttachToNetwork() (domainName string, changeInfo *route53.ChangeInfo, err error) {

  // eipResp, err := awslib.GetNewEIP(p.AWSSession)
  // if err != nil { return anr, err }
  // anr.EIPStatus = eipResp

  domainName = p.GetDomainName()
  comment := fmt.Sprintf("Attaching proxy: %s to network at: %s\n", domainName, p.PublicProxyIp)
  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, domainName, comment, DefaultProxyTTL ,p.AWSSession)

  return domainName, changeInfo, err
}

// Attach a server to the network by pointing the server DNS to this proxy's ip, and use
// this proxy's domain name as the subdomain to attach to.
func (p *Proxy) AttachServerToNetwork(s *Server) (serverFQDN string, changeInfo *route53.ChangeInfo, err error) {

  domainName := p.GetDomainName()
  serverName := makeServerName(s)
  serverFQDN = serverName + "." + domainName
  comment := fmt.Sprintf("Attaching server %s to network at: %s as %s\n", s.Name, p.PublicProxyIp, serverFQDN)
  changeInfo, err = awslib.AttachIpToDNS(p.PublicProxyIp, serverFQDN, comment, DefaultProxyTTL, p.AWSSession)

  return serverFQDN, changeInfo, err
}


func (p *Proxy) NewRcon() (rcon *Rcon, err error) {
  rcon, err = NewRcon(p.PublicProxyIp, p.RconPort.String(), p.RconPassword)  
  if err == nil {
    p.Rcon = rcon
  }
  return rcon, err
}


// These commands assume the following bungee plugins: 
// - BungeeServerManager (/svm)
// - BungeeRcon (which requires BungeeYamler) (responds to Rcon for Bungee)
// - BungeeConfig (which requires bfixlib) (/bconf)
func (p *Proxy) AddServer(s *Server) (err error) {
  rcon, err := p.NewRcon()
  if err != nil { return err }

  fmt.Printf("Connected to RCON: %s:%d\n", rcon.Host, rcon.Port )

  // TODO: Once networking is properly worked out, this should change
  // to a private address.
  name := makeServerName(s)
  command := fmt.Sprintf("svm add %s %s", name, s.PublicServerAddress())
  fmt.Printf("Sending command to rcon: %s\n", command)
  reply, err := rcon.Send(command)
  if err != nil { return err }
  fmt.Printf("Received reply: %s\n", reply)
  log.Debug(logrus.Fields{"reply": reply, "command": command}, "AddServer reply.")

  return err
}

// Add the server as a forced host for this proxy on it's first listener.
// This assumes that there is a server that has already been added
// to the proxy with AddServer or equivelant.
// Create a DNS entry for the server pointing to the IP of the proxy and
// using the subdomain of this poxy.
// That is: if this proxy is proxy.top-level.com, then we create an A DNS record
//  of <server-name>.proxy.top-level.com => Proxy.PublicIPAddress()
func (p *Proxy) ProxyForServer(s *Server) (err error) {

  p.AttachServerToNetwork(s)

  rcon, err := p.NewRcon()
  if err != nil { return err }
  fmt.Printf("Connected to RCON: %s:%d\n", rcon.Host, rcon.Port )

  // TODO: Remove punctuation and othewise make sure this is clean.
  name := makeServerName(s)
  serverDNSName := fmt.Sprintf("%s.%s", name, p.PublicDNSName())
  command := fmt.Sprintf("bconf addForcedHost(%d, \"%s\", \"%s\")", 0, serverDNSName, name)
  fmt.Printf("Sending command to rcon: %s\n", command)
  reply, err := rcon.Send(command)
  fmt.Printf("Received reply: %s\n", reply)
  log.Debug(logrus.Fields{"reply": reply, "command": command}, "ProxyForServer reply.")

  return err
}

// TODO: needs more cleaning (remove punct etc.)
func makeServerName(s *Server) (string) {
  name := strings.Replace(s.Name, " ", "-", -1)
  name = strings.ToLower(name)
  return name
}




