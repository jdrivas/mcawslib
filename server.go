package mclib

import(
  "fmt"
  "time"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  // "github.com/aws/aws-sdk-go/service/s3"
  // "github.com/Sirupsen/logrus"

  // Be Careful ...
  // "awslib"
  "github.com/jdrivas/awslib"

)

// TODO: Likely to want to pull out the AWS 
// into a separate abstration. But for now
// I'm weded to this.
// TODO: Shoud we gather uo the controller in this?
// I expect we should. I don't think we should contemplate servers running
// generally wihtout their controllers. 
type Server struct {
  User string
  Name string
  ServerIp string // Consider wether we need an ARN for an ElasticIP.
  ServerPort int64

  RconPort string
  RconPassword string
  Rcon *Rcon
  
  ArchiveBucket string
  ServerDirectory string

  TaskArn *string
  AWSSession *session.Session
}

func NewServer(userName, serverName, serverIp string, serverPort int64,  rconPort, rconPw, 
  archiveBucket, serverDirectory string, sess *session.Session) (s *Server) {
  s = new(Server)
  s.User = userName
  s.Name = serverName
  s.ServerIp = serverIp
  s.ServerPort = serverPort
  s.RconPort = rconPort
  s.RconPassword = rconPw
  s.ArchiveBucket = archiveBucket
  s.ServerDirectory = serverDirectory
  s.AWSSession = sess
  return s
}


// Proxy versions of these live in proxy.go
const (
  MinecraftServerDefaultArchiveBucket = "craft-cofig-test"
  MinecraftServerContainerName = "minecraft"
  MinecraftControllerContainerName = "minecraft-backup"
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

  LevelKey = "LEVEL"   // World Save name
  LevelDefault = "world"
  ProxyHubLevelDefault = "world"

  OnlineModeKey = "ONLINE_MODE"
  OnlineModeDefault = "true"
  ProxyHubOnlineModeDefault = "false"

  JVMOptsKey = "JVM_OPTS"
  JVMOptsDefault = "-Xmx1024M -Xms1024M"
  ProxyHubJVMOptsDefault = "-Xmx1024M -Xms1024M"

  //TODO:  There is no default for now.
  WorldKey = "WORLD"
  ProxyHubWorldKey = "WORLD"
)


// Get an existing server from the environment.
func GetServer(clusterName, taskArn string, sess *session.Session) (a *Server, err error){
  dtm, err := awslib.GetDeepTasks(clusterName, sess)
  if err != nil { return a, fmt.Errorf("Failed to get Server information: %s", err) }
  dt := dtm[taskArn]
  serverEnv, err := dt.GetEnvironment(MinecraftServerContainerName)
  controllerEnv, err := dt.GetEnvironment(MinecraftControllerContainerName)
  userName := serverEnv[ServerUserKey]
  serverName := serverEnv[ServerNameKey]
  serverIp := dt.PublicIpAddress()
  serverPort, ok := dt.PortHostBinding(MinecraftServerContainerName, ServerPortDefault)
  if !ok {
    serverPort = 0
    log.Error(nil, "Couldn't get server port.", fmt.Errorf("Couldn't get server port."))
  }
  rconPort := serverEnv[RconPortKey]
  rconPW := serverEnv[RconPasswordKey]
  archiveBucket := controllerEnv[ArchiveBucketKey]
  serverDirectory := controllerEnv[ServerLocationKey]
  a = NewServer(userName, serverName, serverIp, serverPort,
    rconPort, rconPW, archiveBucket, serverDirectory, sess)
  return a, err
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
      return nil, fmt.Errorf("Invalid rcon connection paramaters: %s:%s ", s.ServerIp, s.RconPort )
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
  rcon, err = NewRcon(s.ServerIp, s.RconPort, s.RconPassword)  
  if err == nil {
    s.Rcon = rcon
  }
  return rcon, err
}


// Gets a new Rcon connection for the seever. Will retry after waitTime if the connection attempt fails,
// will try up to retry times. Blocks until finished.
func (s *Server) NewRconWithRetry(retries int, waitTime time.Duration) (rcon *Rcon, err error) {
  rcon, err = NewRconWithRetry(s.ServerIp, s.RconPort, s.RconPassword, retries, waitTime)
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
  // return len(s.ServerIp) == 0 || s.RconPort == 0
  return len(s.ServerIp) == 0 || len(s.RconPort) == 0
}

func (s *Server) GoodRcon() (bool) {
  return !s.NoRcon()
}

func (s *Server) newSnapshotPath(when time.Time) (string) {
  return NewSnapshotPath(s.User, s.Name, when)
}

