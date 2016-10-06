package mclib

import(
  "fmt"
  "strings"
  "github.com/aws/aws-sdk-go/aws/session"
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
  if p == nil {
    err = fmt.Errorf("Error: couldn't find proxy with name: %s", proxyName)
  }
  return p, err
}

// TODO: THIS IS IMPORTANT. We need to check the DNS to see if we're 
// currently attached to tne network or not. Suggested updates 
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


// This can be a little expensive. It makes 4 calls to AWS.
// func (p *Proxy) GetDeepTask() (dt *awslib.DeepTask, err error) {
//   return awslib.GetDeepTask(p.ClusterName, p.TaskArn, p.AWSSession)
// }

func (p *Proxy) NewRcon() (rcon *Rcon, err error) {
  rcon, err = NewRcon(p.PublicProxyIp, p.RconPort.String(), p.RconPassword)  
  if err == nil {
    p.Rcon = rcon
  }
  return rcon, err
}

// Get Rcon returns a working rcon connection and 
// will reuse an existing one if possible
func (p *Proxy) GetRcon() (rcon *Rcon, err error) {
  if p.Rcon == nil {
    _, err = p.NewRcon()
  }
  return p.Rcon, err 
}


// These commands assume the following bungee plugins: 
// - BungeeConfig (which requires bfixlib) (/bconf)
// TODO: build out a library that uses these commands directly.
// way too much repetition here.
// morevoer we should abstract away the Rcon connection.
// it works, but longer term this has to be gotten rid of for 
// something more secure and robust.

// Add the server to the proxy. Access to server is availble when connected
// through the proxy.
func (p *Proxy) AddServerAccess(s *Server) (err error) {
  f:= logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
  }
  log.Info(f, "Adding server access to proxy.")

  rcon, err := p.GetRcon()
  if err != nil { return err }

  // TODO: Once networking is properly worked out, this should change
  // to a private address.
  motd := fmt.Sprintf("%s hosted by %s in the %s neighborhood.", s.Name, s.User, s.Name)
  command :=  fmt.Sprintf("bconf addServer(\"%s\", \"%s\", \"%s\", false)",
    s.Name, motd, s.PublicServerAddress())

  reply, err := rcon.Send(command)
  f["command"] = command
  f["reply"] = reply
  if err != nil { 
    log.Error(f, "Remore addServer failed.", err)
    return err 
  }
  // fmt.Printf("Received reply: %s\n", reply)
  log.Info(f, "Remote addServer reply.")

  return err
}


// TODO: Need to remove the rcon stuff here, and find a better way to 
// determine success.
// Remove access to the server for this proxy. The server is no longer
// reachable through the proxy.
func (p *Proxy) RemoveServerAccess(s *Server) (error) {

  f:= logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
  }
  log.Info(f, "Removing server from  proxy (proxy access and forced-host)")

  found, err := p.isServerProxied(s)
  if err != nil { return err }
  if !found { 
    err = fmt.Errorf("Server: %s not with this proxy: %s.", s.Name, p.Name)
    log.Error(f,"Server not removed.", err)
    return err
  }

  rcon, err := p.GetRcon()
  if err != nil { 
    log.Error(f,"Server not removed.", err)
    return err 
  }

  // Remove the server access from the proxy.
  command :=  fmt.Sprintf("bconf remServer(\"%s\")", s.Name)
  reply, err := rcon.Send(command)
  f["command"] = command
  f["reply"] = reply
  if err != nil { 
    log.Error(f, "Proxy access to Server remove: Remote removeServer errored.", err)
  } else {
    log.Info(f, "Proxy access to server remove: complete.")
  }

  return err
}

// Add the server as a forced host for this proxy on it's first listener.
// This assumes that there is a server that has already been added
// to the proxy with AddServer or equivelant.
// Create a DNS entry for the server pointing to the IP of the proxy and
// using the subdomain of this poxy.
// That is: if this proxy is proxy.top-level.com, then we create an A DNS record
//  of <server-name>.proxy.top-level.com => Proxy.PublicIPAddress()
func (p *Proxy) StartProxyForServer(s *Server) (err error) {

  f:= logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
  }
  log.Info(f, "Adding proxy as a network-proxy for server.")

  serverFQDN, err := p.ProxiedServerFQDN(s)
  if err != nil {
    log.Error(f, "Failed to obtain server address from DNS. Proxy not started for Server", err)
    return err
  }
  f["serverFQDN"] = serverFQDN

  rcon, err := p.GetRcon()
  if err != nil { 
    log.Error(f, "Failed to get an RCON connection. Proxy not started for Server", err)
    return err 
  }

  // serverFQDN := fmt.Sprintf("%s.%s", s.DNSName(), proxyFQDN)
  command := fmt.Sprintf("bconf addForcedHost(%d, \"%s\", \"%s\")", 0, serverFQDN, s.Name)
  reply, err := rcon.Send(command)
  f["command"] = command
  f["reply"] = reply
  log.Info(f, "Remote addForcedHost reply.")

  return err
}


// Stop proxying for the Server.
// Assumes that the server DNS record is still pointing to the proxy.
// Wil
func (p* Proxy) StopProxyForServer(s *Server) (error) {

  f := logrus.Fields{
    "proxy": p.Name, "server": s.Name, "user": s.User,
  }

  rcon, err := p.GetRcon()
  if err != nil { 
    log.Error(f,"Server not removed.", err)
    return err 
  }

  serverFQDN, err := p.ProxiedServerFQDN(s)
  if err != nil { 
    serverFQDN = p.attachedServerFQDN(s)
    log.Error(f, "Failed to get proxy name from DNS. Will try to remove server forced host anyway.", err)
  }
  f["serverFQDN"] = serverFQDN

  command := fmt.Sprintf("bconf remForcedHost(%d, \"%s\")", 0, serverFQDN)
  reply, err := rcon.Send(command)
  f["command"] = command
  f["reply"] = reply
  if err == nil {
    log.Info(f, "Remote remove-forced-host completed.")
  } else {
    log.Error(f, "Failed on remote remove-forced-host: will try to remote remove-server.", err)
  }

  return err
}

func (p *Proxy) isServerProxied(s *Server) (bool, error) {
  serverNames, err := p.ServerNames()
  if err != nil { return false, err }
  found := false
  for _, n := range serverNames {
    if n == s.Name {
      found = true
      break
    }
  }
  return found, err
}

// The names of the servers that are currently available through this proxy.
// Yes, of course I know this is not the way to do this.
// For now it's all I have however ......
func (p *Proxy) ServerNames() ([]string, error) {
  ns := []string{"Error-Getting-Server-Names"}
  rcon, err := p.GetRcon()
  if err != nil { return ns, err }

  command := fmt.Sprintf("bconf getServers().getKeys()")
  reply, err := rcon.Send(command)
  if err != nil { return ns, err }

  reply = strings.Trim(reply, "[] \n")
  names := strings.Split(reply, ",")
  for i, n := range names {
    names[i] = strings.Trim(n, " ")
  }
  return names, nil
}


