package mclib

import(
  "fmt"
  "strconv"
  "time"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  // "github.com/Sirupsen/logrus"

  // Be Careful ...
  // "awslib"
  "github.com/jdrivas/awslib"

)

// Proxy versions of these live in proxy.go
const (
  DefaultVanillaServerTaskDefinition = "minecraft-ecs"
  DefaultProxiedServerTaskDefinition = "bungee-spigot"
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

  if !ok {
    err = fmt.Errorf("Error finding server for %s/%s : %s", clusterName, taskArn, err)
  }
  return s, err
}

func GetServerForName(n, cluster string, sess *session.Session) (s *Server, err error) {
  servers, err := GetServers(cluster, sess)
  if err != nil {return s, err}
  for _, srv := range servers {
    if srv.Name == n {
      s = srv
      break
    }
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
    serverPort := Port(0)
    if port, ok := dt.PortHostBinding(MinecraftServerContainerName, ServerPortDefault); ok {
      serverPort = Port(port)
    }
    rconPort := Port(0)
    if port, ok := dt.PortHostBinding(MinecraftControllerContainerName, RconPortDefault); ok {
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

// Does lookups in DNS to find the DNS for this server.
// OR at least it should/will.
func (s *Server) DNSAddress() (string) {
  return s.PublicServerIp + ":" + s.ServerPort.String()
}

func (s *Server) PublicServerAddress() (string) {
  return s.PublicServerIp + ":" + s.ServerPort.String()
}

func (s *Server) RconAddress() (string) {
  return s.PrivateServerIp + ":" + s.RconPort.String()
}

func (s *Server) checkForNullTask() (bool) {
  return s == nil || s.DeepTask == nil || s.DeepTask.Task == nil
}

func (s *Server) ServerTaskStatus() (string) {
  if s.checkForNullTask() { return "---" }
  return *s.DeepTask.Task.LastStatus
}

func (s *Server) ServerContainer() (*ecs.Container, bool) {
  return getContainerFromNames(GetServerContainerNames(), s.DeepTask)
}

func (s *Server) ControllerContainer() (*ecs.Container, bool) {
  return getContainerFromNames(GetControllerContainerNames(), s.DeepTask)
}

// Returns the containers environment (Update env actually.).
// Ok is false if the Server container couldn't be found.
func (s *Server) ServerEnvironment() (cenv map[string]string, ok bool) {
  if s.checkForNullTask() { return cenv, ok }
  return s.DeepTask.EnvironmentFromNames(GetServerContainerNames())
}

func (s *Server) ControllerEnvironment() (cenv map[string]string, ok bool) {
  if s.checkForNullTask() { return cenv, ok }
  return s.DeepTask.EnvironmentFromNames(GetControllerContainerNames())
}

func (s *Server) ServerContainerStatus() string {
  status := "---"
  if s.checkForNullTask()  { return status }
  c, ok := s.ServerContainer()
  if ok {
    status = *c.LastStatus
  }
  return status
}

func (s *Server) ControllerContainerStatus() string {
  status := "---"
  if s.checkForNullTask()  { return status }
  c, ok := s.ControllerContainer()
  if ok {
    status = *c.LastStatus
  }
  return status
}

func (s *Server) UptimeString() (string) {
  return s.DeepTask.UptimeString()
}

func (s *Server) Uptime() (time.Duration, error) {
  return s.DeepTask.Uptime()
}

func (s *Server) CraftType() (string) {
  env, ok := s.ServerEnvironment()
  if !ok { return "<unknown-type>" }
  serverType, ok := env[TypeKey]
  if !ok { serverType = "Vanila" }
  return serverType
}

// Will take snapshot of the server and publish it to the S3 bucket.
// snapshots are stored
//         bucket:/<Server.User>/<Server.Name>/snapshots/<time.Now()_ansi-time-string>-<Server.User>-<Server.Name>-snapshot.zip
// Zip files are used because that's the standard in minecraft land.
// 
// If serverIP or rconPort are not nil, then an rcon connection to 
// the server will be made to issue save-all then save-off before
// the snapshot is taken and save-on after once the snapshot has
// been ade but before it's been published to S3. THIS IS NOT RECOMMENDED FOR PRDOCUTION.
func (s *Server) SnapshotAndPublish() ( resp *PublishedArchiveResponse, err error) {

  var rcon  *Rcon
  if s.GoodRcon() && !s.HasRconConnection() {
    rcon, err = s.NewRcon()
    if err != nil { return resp, fmt.Errorf("Can't create rcon connection for snapshot snapshot: %s", err)}
  }

  resp, err = s.archiveAndPublish(rcon)
  return resp, err
}

// If we don't already have an RCON, will call NewRconWithRetry to get one.
func (s *Server) SnapshotAndPublishWithRetry(retries int, waitTime time.Duration) (resp *PublishedArchiveResponse, err error) {

  if !s.HasRconConnection() {
    if s.NoRcon() { 
      return nil, fmt.Errorf("Invalid rcon connection paramaters: %s:%s ", s.PublicServerIp, s.RconPort )
    }
    if len(s.RconPassword) == 0 {
      return nil, fmt.Errorf("No rcon password.")
    }

    _, err = s.NewRconWithRetry(retries, waitTime)
    if err != nil { return nil, err }
  }

  resp, err = s.archiveAndPublish(s.Rcon)
  return resp, err
}

func (s *Server) GetSnapshotList() (snaps []Archive, err error) {
  am, err := GetArchives(s.User, s.Name, s.AWSSession)
  if err == nil {
    snaps = am.GetSnapshots(s.User)
  }
  return snaps, err
}

func (s *Server) archiveAndPublish(rcon *Rcon) (resp *PublishedArchiveResponse, err error) {
  resp, err = ArchiveAndPublish(rcon, s.ServerDirectory, s.ArchiveBucket, 
    s.newSnapshotPath(time.Now()), s.AWSSession)
  return resp, err
}

func (s *Server) NewRcon() (rcon *Rcon, err error) {
  rcon, err = NewRcon(s.PublicServerIp, s.RconPort.String(), s.RconPassword)  
  if err == nil {
    s.Rcon = rcon
  }
  return rcon, err
}


// Gets a new Rcon connection for the seever. Will retry after waitTime if the connection attempt fails,
// will try up to retry times. Blocks until finished.
func (s *Server) NewRconWithRetry(retries int, waitTime time.Duration) (rcon *Rcon, err error) {
  rcon, err = NewRconWithRetry(s.PublicServerIp, s.RconPort.String(), s.RconPassword, retries, waitTime)
  if err == nil {
    s.Rcon = rcon
  }
  return rcon, err
}

func (s *Server) HasRconConnection() (bool) {
  if s.Rcon == nil {
    return false
  }
  return s.Rcon.HasConnection()
}

func (s *Server) NoRcon() (bool) {
  return len(s.PublicServerIp) == 0 || s.RconPort == 0
  // return len(s.ServerIp) == 0 || len(s.RconPort) == 0
}

func (s *Server) GoodRcon() (bool) {
  return !s.NoRcon()
}

func (s *Server) newSnapshotPath(when time.Time) (string) {
  return NewSnapshotPath(s.User, s.Name, when)
}

