package mclib

import(
  "fmt"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/route53"

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
  ProxyPort int64
  RconPort int64
  RconPassword string
  Rcon *Rcon

  TaskArn string
  AWSSession *session.Session
}

const (
  BungeeProxyServerContainerName = "bungee"
  BungeeProxyHubServerContainerName = "minecraft-hub"
  BungeeProxyHubControllerContainerName = "minecraft-control"
)

func NewProxy(name, clusterName, publicIp, privateIp, taskArn, rconPw string,
  proxyPort, rconPort int64, sess *session.Session ) (p *Proxy) {

  p = new(Proxy)
  p.Name = name
  p.ClusterName = clusterName
  p.PublicProxyIp = publicIp
  p.PrivateProxyIp = privateIp
  p.ProxyPort = proxyPort
  p.RconPort = rconPort
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

func GetProxyByName(clusterName, proxyName string, sess *session.Session) (p *Proxy, err error) {
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

// TODO: THIS IS IMPORTANT. We need to check the DNS to see if we're currently attached to tne 
// network or not.  Suggested updates include: add a new field to the Proxy struct which is the
// actual DNS address for this proxy and have this function AND ONLY this function fill it out.
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

func (p *Proxy) GetDomainName() (dn string) {
  // dn = p.Name + ".hood.momentlabs.io"
  dn = p.Name + ".momentlabs.io"
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
  if err != nil { return domainName, changeInfo, err }

  return domainName, changeInfo, err
}









