package mclib

import(
  "fmt"
  "strconv"
  "time"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/Sirupsen/logrus"

  // Be Careful ...
  // "awslib"
  "github.com/jdrivas/awslib"

)

// Proxy versions of these live in proxy.go
const (
  DefaultVanillaServerTaskDefinition = "minecraft-ecs"
  DefaultProxiedServerTaskDefinition = "bungee-spigot-nc"
  DefaultServerTaskDefinition = DefaultProxiedServerTaskDefinition

  MinecraftServerContainerName = "minecraft"
  MinecraftControllerContainerName = "minecraft-backup"
)

const (
  MinecraftServerDefaultArchiveBucket = "craft-cofig-test"
)

//
// Container Environment Variabls
//


// We're keeping all conatiner environment variable keys here.
// Unproxied Server/Controller
// Proxy/Hub/Controller
// Proxied-Server


// These are container env variables that are used
// to manage a collection of containers as opposed
// to the server variables designed really to configure
// the server.
const (

  // For all containers
  RoleKey = "CONTAINER_ROLE"
  CraftServerRole = "CraftServer"
  CraftControllerRole = "CraftController"
  CraftProxyRole = "CraftProxy"
  CraftHubServerRole = "CraftHubServer"

  // For Craft Servers
  ServerNameKey = "SERVER_NAME"
  ServerUserKey = "SERVER_USER"
  BackupRegionKey = "CRAFT_BACKUP_REGION"
  ArchiveRegionKey = "CRAFT_ARCHIVE_REGION"
  ArchiveBucketKey = "ARCHIVE_BUCKET"
  ServerLocationKey = "SERVER_LOCATION"
  ClusterNameKey = "CLUSTER_NAME"
  // This is where server files are located relative to 
  // the controller.
  ServerLocationDefault = "." 

  // For Proxys

)

const(
  ServerPortDefault = 25565
  ServerPortDefaultString = "25565"
  ProxyPortDefault = 25577
  ProxyPortDefaultString = "25577"
  QueryPortDefault = 25565
  QueryPortDefaultString = "25565"
  RconPortDefault = 25575
  RconPortDefaultString = "25575"
)


// Minecraft Server Config Container Environment Variables
// and Default Values.
// Config file strings are not in here yet!
const (

  OpsKey = "OPS"
  // This is how it is usually done.
  // OpsDefault = userName
  ModeKey = "MODE"
  ModeDefault = "creative"
  ProxyHubModeDefault = "creative"

  ViewDistanceKey = "VIEW_DISTANCE"
  ViewDistanceDefault = "10"
  ProxyHubViewDistanceDefault = "10"

  SpawnAnimalsKey = "SPAWN_ANIMALS"
  SpawnAnimalsDefault = "true"
  ProxyHubSpawnAnimalsDefault = "false"

  SpawnMonstersKey = "SPAWN_MONSTERS"
  SpawnMonstersDefault = "false"
  ProxyHubSpawnMonstersDefault = "false"

  SpawnNPCSKey = "SPAWN_NPCS"
  SpawnNPCSDefault = "true"
  ProxyHubSpawnNPCSDefault = "false"

  ForceGameModeKey = "FORCE_GAMEMODE"
  ForceGameModeDefault = "true"
  ProxyHubForceGameModeDefault = "true"

  GenerateStructuresKey = "GENERATE_STRUCTURES"
  GenerateStructuresDefault = "true"
  ProxyHubGenerateStructuresDefault = "false"

  AllowNetherKey = "ALLOW_NETHER"
  AllowNetherDefault = "true"
  ProxyHubAllowNetherDefault = "false"

  MaxPlayersKey = "MAX_PLAYERS"
  MaxPlayersDefault = "20"
  ProxyHubMaxPlayersDefault = "20"

  QueryKey = "QUERY"
  QueryDefault = "true"
  ProxyHubQueryDefault = "true"

  QueryPortKey = "QUERY_PORT"
  // QueryPortDefault = QueryPortDefaultString
  ProxyHubQueryPortDefault = QueryPortDefaultString

  EnableRconKey = "ENABLE_RCON"
  EnableRconDefault = "true"
  ProxyHubEnableRconDefault = "true"

  RconPortKey = "RCON_PORT"
  // RconPortDefault = RconPortDefaultString
  ProxyHubRconPortDefault = RconPortDefaultString

  RconPasswordKey = "RCON_PASSWORD"
  RconPasswordDefault = "testing"   // TODO: NO NO NO NO NO NO NO NO NO NO
  ProxyHubRconPasswordDefault = "testing"   // TODO: NO NO NO NO NO NO NO NO NO NO
  ProxyRconPasswordDefault = "testing" // TODO: NO NO NO NO NO NO NO NO NO NO

  MOTDKey = "MOTD"
  // This is how it's usually done:
  // MOTDDefault = fmt.Sprintf("A neighborhood kept by %s.", userName)
  PVPKey = "PVP"
  PVPDefault = "false"
  ProxyHubPVPDefault = "false"

  TypeKey = "TYPE" 
  // Values can be:
  // Vannila is an empty server.
  BukkitTypeValue = "BUKKIT"
  SpigotTypeValue = "SPIGOT"
  ForgeTypeValue = "FORGE"
  PapertypeValue = "PAPER"

  LevelKey = "LEVEL"   // World Save name
  LevelDefault = "world"
  ProxyHubLevelDefault = "world"

  OnlineModeKey = "ONLINE_MODE"
  OnlineModeDefault = "true"
  ProxiedServerOnlineModeDefault = "false"
  ProxyHubOnlineModeDefault = "false"

  JVMOptsKey = "JVM_OPTS"
  JVMOptsDefault = "-Xmx1024M -Xms1024M"
  ProxyHubJVMOptsDefault = "-Xmx1024M -Xms1024M"

  //TODO:  There is no default for now.
  WorldKey = "WORLD"
  ProxyHubWorldKey = "WORLD"
)


type Port int64

func (p Port) String() (string) {
  return strconv.FormatInt(int64(p), 10)
}

func NewPort(ps string) (Port, error) {
  p, err := strconv.ParseInt(ps, 10, 64)
  return Port(p), err
}

// TODO: Likely to want to pull out the AWS 
// into a separate abstration. But for now
// I'm weded to this.
// TODO: Shoud we gather uo the controller in this?
// I expect we should. I don't think we should contemplate servers running
// generally wihtout their controllers. 
type Server struct {
  User string
  Name string
  ClusterName string

  PublicServerIp string // Consider wether we need an ARN for an ElasticIP.
  PrivateServerIp string  

  ServerPort Port
  RconPort Port

  RconPassword string
  Rcon *Rcon
  
  ArchiveBucket string
  ServerDirectory string

  TaskArn *string
  DeepTask *awslib.DeepTask
  AWSSession *session.Session
}

func GetServers(clusterName string, sess *session.Session) (s []*Server, err error) {
  s = make([]*Server, 0)
  dtm, err := awslib.GetDeepTasks(clusterName, sess)
  if err != nil {return s, err}

  for _, dt := range dtm {
    server, ok := GetServerFromTask(dt, sess)
    if ok {
      s = append(s, server)
    }
  }
  return s, err
}

// Get an existing server from the environment.
func GetServer(clusterName, taskArn string, sess *session.Session) (s *Server, err error){
  dtm, err := awslib.GetDeepTasks(clusterName, sess)
  if err != nil { return s, fmt.Errorf("Failed to get Server information: %s", err) }
  dt := dtm[taskArn]
  s, ok := GetServerFromTask(dt, sess)
  s.ClusterName = clusterName

  if !ok {
    err = fmt.Errorf("Error finding server for %s/%s : %s", clusterName, taskArn, err)
  }
  return s, err
}

// Get a server (probably recently launced) but wait until the task is running, ensuring
// we have things like allocated ports.
func GetServerWait(clusterName, taskArn string, sess *session.Session) (s *Server, err error) {
  err = awslib.WaitForTaskRunning(clusterName, taskArn, sess)
  if err != nil { return s, fmt.Errorf("Failed to wait for task: s", err) }
  return GetServer(clusterName, taskArn, sess)
}

func GetServerFromName(n, cluster string, sess *session.Session) (s *Server, err error) {
  servers, err := GetServers(cluster, sess)
  if err != nil {return s, err}
  for _, srv := range servers {
    if srv.Name == n {
      s = srv
      break
    }
  }
  if s == nil {
    err = fmt.Errorf("Error: coudln't find server with name: %s", n)
  }
  return s, err
}

// See the similar note over at proxy.go/GetProxyFromTask()
// TODO: THIS IS IMPORTANT We are currently determining wether or not a task is a Server task
// by looking for the presence of a particularly named container. It's unclear what kind
// of trouble this will cause. At least we know we have to coordinate closely with the 
// task-definitions.
func GetServerFromTask(dt *awslib.DeepTask, sess *session.Session) (s *Server, ok bool) {
  serverEnv, ok := getServerEnv(dt)
  controllerEnv, _ := getControllerEnv(dt)
  if ok {

    sc, ok := getContainerFromNames(GetServerContainerNames(), dt)
    if !ok { return nil, false }
    serverPort := Port(0)
    if port, ok := dt.PortHostBinding(*sc.Name, ServerPortDefault); ok {
      serverPort = Port(port)
    }
    rconPort := Port(0)
    if port, ok := dt.PortHostBinding(*sc.Name, RconPortDefault); ok {
      rconPort = Port(port)
    }
    s = &Server{
      User: serverEnv[ServerUserKey], 
      Name: serverEnv[ServerNameKey],
      ClusterName: dt.ClusterName(), 
      PublicServerIp: dt.PublicIpAddress(), 
      PrivateServerIp: dt.PrivateIpAddress(),
      ServerPort: serverPort,
      RconPort: rconPort,
      RconPassword: serverEnv[RconPasswordKey],
      ArchiveBucket: controllerEnv[ArchiveBucketKey],
      ServerDirectory: controllerEnv[ServerLocationKey],
      TaskArn: dt.Task.TaskArn,
      DeepTask: dt,
      AWSSession: sess,
    }
  } else {
    fmt.Printf("Ddin't return controller.\n")
  }
  return s, ok
}

func GetServerContainerNames() []string {
  return []string{MinecraftServerContainerName, BungeeProxyHubServerContainerName,}
}

func GetControllerContainerNames() []string {
  return []string{MinecraftControllerContainerName, BungeeProxyHubControllerContainerName,}
}

func getServerEnv(dt *awslib.DeepTask) (map[string]string, bool) {
  return dt.EnvironmentFromNames(GetServerContainerNames())
}

func getControllerEnv(dt *awslib.DeepTask) (map[string]string, bool) {
  return dt.EnvironmentFromNames(GetControllerContainerNames())
}

// Returns we find in the list.
func getContainerFromNames(containers []string, dt*awslib.DeepTask) (c *ecs.Container, ok bool) {
  for _, cn := range containers {
    cntr, k := dt.GetContainer(cn)
    if k {
      c = cntr
      ok = true
      break
    }
  }
  return c, ok
}



// Returns the IP and the Port.
func (s *Server) PublicServerAddress() (string) {
  return s.PublicServerIp + ":" + s.ServerPort.String()
}

// Returned the IP and the port
func (s *Server) RconAddress() (string) {
  return s.PrivateServerIp + ":" + s.RconPort.String()
}

// Convenience to check the DeepTask and DeepTask.Task for non-null
func (s *Server) checkForNullTask() (bool) {
  return s == nil || s.DeepTask == nil || s.DeepTask.Task == nil
}

// Gets the LastStaus from DeepTask.Task
func (s *Server) ServerTaskStatus() (string) {
  if s.checkForNullTask() { return "---" }
  return *s.DeepTask.Task.LastStatus
}

// The container the Server is running in.
func (s *Server) ServerContainer() (*ecs.Container, bool) {
  return getContainerFromNames(GetServerContainerNames(), s.DeepTask)
}

// The Container the ServerController is running in.
func (s *Server) ControllerContainer() (*ecs.Container, bool) {
  return getContainerFromNames(GetControllerContainerNames(), s.DeepTask)
}

// Returns the Server containers environment (Update env actually).
// Ok is false if the Server container couldn't be found.
func (s *Server) ServerEnvironment() (cenv map[string]string, ok bool) {
  if s.checkForNullTask() { return cenv, ok }
  return s.DeepTask.EnvironmentFromNames(GetServerContainerNames())
}

// Returns the Controller Containers environment (Update env actually).
// Ok is false if the container couldn't be found.
func (s *Server) ControllerEnvironment() (cenv map[string]string, ok bool) {
  if s.checkForNullTask() { return cenv, ok }
  return s.DeepTask.EnvironmentFromNames(GetControllerContainerNames())
}

// Status of the Server Container (ecs LastStatus)
func (s *Server) ServerContainerStatus() string {
  status := "---"
  if s.checkForNullTask()  { return status }
  c, ok := s.ServerContainer()
  if ok {
    status = *c.LastStatus
  }
  return status
}

// Status of the Controller Container (ecs Last Status)
func (s *Server) ControllerContainerStatus() string {
  status := "---"
  if s.checkForNullTask()  { return status }
  c, ok := s.ControllerContainer()
  if ok {
    status = *c.LastStatus
  }
  return status
}

// Uptime  of the server task nicely formatted (awslib.UptimeString())
func (s *Server) UptimeString() (string) {
  return s.DeepTask.UptimeString()
}

// Uptime of the server task.
func (s *Server) Uptime() (time.Duration, error) {
  return s.DeepTask.Uptime()
}

// The type of craft server running, from the container's environment.
func (s *Server) CraftType() (string) {
  env, ok := s.ServerEnvironment()
  if !ok { return "<unknown-type>" }
  serverType, ok := env[TypeKey]
  if !ok { serverType = "Vanila" }
  return serverType
}

// Convenience to provide fields all filed out for logging actions on this server.
func (s *Server) LogFields() (logrus.Fields) {
  cluster := "<none>"
  if s.ClusterName != "" { cluster = s.ClusterName }
  arn := "<none>"
  if s.TaskArn != nil { arn = *s.TaskArn }
  f := make(logrus.Fields)
  f["userName"] = s.User
  f["serverName"] = s.Name
  f["cluster"] = cluster
  f["arn"] = arn
  return f
}


