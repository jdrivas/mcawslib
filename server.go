package mclib

import(
  "fmt"
  "strconv"
  "strings"
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
  // DefaultProxiedServerTaskDefinition = "bungee-spigot-nc"
  DefaultProxiedServerTaskDefinition = "craft-server"
  DefaultServerTaskDefinition = DefaultProxiedServerTaskDefinition

  // TODO: Find a better way to do this. Probably using role key.
  // These are critical container names that are used
  // by the system to differentiate among environments
  // from a container.
  // THIS IS LIKELY A MISTAKE.
  MinecraftServerContainerName = "minecraft"
  MinecraftControllerContainerName = "minecraft-backup"
)

const (
  MinecraftServerDefaultArchiveBucket = "craft-config-test"
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
  TaskArnKey = "TASK_ARN"
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

  // Name
  User string
  Name string

  // Network
  PublicServerIp string // Consider wether we need an ARN for an ElasticIP.
  PrivateServerIp string  
  ServerPort Port

  // Rcon
  RconPort Port
  RconPassword string
  Rcon *Rcon
  
  // Archive
  ArchiveBucket string
  ServerDirectory string

  // Task/Container
  ClusterName string
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

  dt := dtm[awslib.ShortArnString(&taskArn)]
  if dt == nil {
    err = fmt.Errorf("Failed to get server, couldn't find DeepTask in returned map for taskArn: %s", taskArn)
    log.Error(logrus.Fields{"cluster": clusterName, "taskArn": taskArn,}, "Failed to get server.", err)
    return s, err
  }

  s, ok := GetServerFromTask(dt, sess)
  if !ok {
    err = fmt.Errorf("Error finding server for %s/%s : %s", clusterName, taskArn, err)
  } else {
    f := s.LogFields()
    f["clusterNameArg"] = clusterName
    f["taskArnArg"] = taskArn
    log.Debug(f, "Retrieved server from task.")
  }
  return s, err
}

// Get a server (probably recently launced) but wait until the task is running, ensuring
// we have things like allocated ports.
func GetServerWait(clusterName, taskArn string, sess *session.Session) (s *Server, err error) {
  err = awslib.WaitForTaskRunning(clusterName, taskArn, sess)
  if err != nil { return s, fmt.Errorf("Failed to wait for task: %s", err) }
  return GetServer(clusterName, taskArn, sess)
}

// Returns a server for the serverName in the cluster.
// Will error if it can't find one, or if there are more than one with the same name.
func GetServerFromName(n, cluster string, sess *session.Session) (s *Server, err error) {
  servers, err := GetServers(cluster, sess)
  if err != nil {return s, err}
  for _, srv := range servers {
    if srv.Name == n {
      if s == nil {
        s = srv
      } else {
        return s, fmt.Errorf("Found more than one server with name %s", n)
      }
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
  log.Debug(logrus.Fields{"taskArn": dt.Task.TaskArn}, "Getting a server from a DeepTask.")
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
  } 
  return s, ok
}

// Stops the task associated with this server.
func (s *Server) Terminate() (taskArn string, err error) {
  taskArn = *s.TaskArn
  _, err = awslib.StopTask(s.ClusterName, *s.TaskArn, s.AWSSession)

  f := s.LogFields()
  f["operation"] = "TermminateServer"
  if err != nil {
    log.Error(f, "Error terminating server.", err)
  } else {
    log.Info(f, "Terminating server.")
  }

  return taskArn, err
}

// Default address for server.
// This should a VPN address as opposed
// to a publically accessible address.
// Returns the IP and the port.
func (s *Server) ServerAddress() (string) {
  return s.PrivateServerIp + ":" + s.ServerPort.String()
}

// Returned the IP and the port
func (s *Server) RconAddress() (string) {
  return s.PrivateServerIp + ":" + s.RconPort.String()
}

// TODO: Should this return an error or some other
// way to note that there is no server address available
// if we are not configured to expose the public address?
// Returns the IP and the Port.
func (s *Server) PublicServerAddress() (string) {
  return s.PublicServerIp + ":" + s.ServerPort.String()
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

// StartedAt - when the server started.
func (s *Server) StartedAtString() (string) {
  return s.DeepTask.StartedAtString()
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
  if !ok { serverType = "Vanilla" }
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
  f["serverAddress"] = s.ServerAddress()
  f["rconAddress"] = s.RconAddress()
  return f
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

// Server Sorting Interface
// To create a new sorting category use ByStartAt as a template, 
// just replace the less function.
//
// To use:
// servers []*Server 
// sort.Srot(ByStartAt(servers))

type serverSort struct {
  s []*Server
  less func(i,j *Server) (bool)
}
func (ss serverSort) Len() int { return len(ss.s) }
func (ss serverSort) Swap(i, j int) { ss.s[i], ss.s[j] = ss.s[j], ss.s[i] }
func( ss serverSort) Less(i, j int) bool { return ss.less( ss.s[i], ss.s[j]) }

func ByStartAt(servers []*Server) (serverSort) {
  return serverSort{
    s: servers,
    less: func(si, sj *Server) (bool) {
      ti := si.DeepTask.Task.StartedAt
      tj := sj.DeepTask.Task.StartedAt 
      // From time to time we get nil times,
      // usually due to querying the interface before the container has started.
      switch {
      case ti == nil && tj == nil:
        r := strings.Compare(fmt.Sprintf("%s", si),fmt.Sprintf("%s", sj))
        switch {
        case  r <= 0: return true
        case r > 0: return false
      }
      case ti == nil: return true
      case tj == nil: return false
      }
      return si.DeepTask.Task.StartedAt.Before(*sj.DeepTask.Task.StartedAt)
    },
  }
}
