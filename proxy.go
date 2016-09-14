package mclib

import(
  "fmt"
  "github.com/aws/aws-sdk-go/aws/session"

  // "awslib"
  "github.com/jdrivas/awslib"
)



// TODO: Is there state that we need to save on the 
// Proxy? Probably as we addd plugins.
type Proxy struct {
  Name string

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

func NewProxy(name, publicIp, privateIp, taskArn, rconPw string,
  proxyPort, rconPort int64, sess *session.Session ) (p *Proxy) {

  p = new(Proxy)
  p.Name = name
  p.PublicProxyIp = publicIp
  p.PrivateProxyIp = privateIp
  p.ProxyPort = proxyPort
  p.RconPort = rconPort
  p.RconPassword = rconPw
  p.TaskArn = taskArn
  p.AWSSession = sess

  return p
}

func GetProxy(clusterName, taskArn string, sess *session.Session) (*Proxy, error) {
  dt, err := awslib.GetDeepTask(clusterName, taskArn, sess)
  if err != nil { return nil, err }
  proxyEnv, err := dt.GetEnvironment(BungeeProxyServerContainerName)
  proxyPort := int64(0)
  if p, ok := dt.PortHostBinding(BungeeProxyServerContainerName, ProxyPortDefault); ok {
    proxyPort = p
  }
  rconPort := int64(0)
  if p, ok := dt.PortHostBinding(BungeeProxyServerContainerName, RconPortDefault); ok {
    rconPort = p
  }

  proxy := NewProxy(
    proxyEnv[ServerNameKey], dt.PublicIpAddress(), dt.PrivateIpAddress(), taskArn,
    proxyEnv[RconPasswordKey], proxyPort, rconPort, sess)

  return proxy, nil
}

func (p *Proxy) PublicIpAddress() (string) {
  return fmt.Sprintf("%s:%d", p.PublicProxyIp, p.ProxyPort)
}

func (p *Proxy) RconAddress() (string) {
  return fmt.Sprintf("%s:%d", p.PrivateProxyIp, p.RconPort)
}



// func GetProxyFromTask(dt *awslib.DeepTask, sess *session.Session) (*Proxy) {
  // 
// }