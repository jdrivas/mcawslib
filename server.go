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
type Server struct {
  User string
  Name string
  ServerIp string

  RconPort string
  RconPassword string
  Rcon *Rcon
  
  ArchiveBucket string
  ServerDirectory string

  TaskArn *string
  AWSSession *session.Session
}

func NewServer(userName, serverName, serverIp, rconPort, rconPw, 
  archiveBucket, serverDirectory string, sess *session.Session) (s *Server) {
  s = new(Server)
  s.User = userName
  s.Name = serverName
  s.ServerIp = serverIp
  s.RconPort = rconPort
  s.RconPassword = rconPw
  s.ArchiveBucket = archiveBucket
  s.ServerDirectory = serverDirectory
  s.AWSSession = sess
  return s
}

// System Defaults
const (
  MinecraftServerDefaultArchiveBucket = "craft-cofig-test"
  MinecraftServerContainerName = "minecraft"
  MinecraftControllerContainerName = "minecraft-backup"
)



// Container Environment Variabls
const (
  // TODO: This needs to move somewhere (probaby mclib).
  // But until that get's done. these are copied over into
  // craft-config. Not very safe
  ServerUserKey = "SERVER_USER"
  ServerNameKey = "SERVER_NAME"
  BackupRegionKey = "CRAFT_BACKUP_REGION"
  ArchiveRegionKey = "CRAFT_ARCHIVE_REGION"
  ArchiveBucketKey = "ARCHIVE_BUCKET"
  ServerLocationKey = "SERVER_LOCATION"
  ServerLocationDefault = "." // This is where the server is located relative to the controller.
)

// Minecraft Server Config Container Environment Variables
// and Default Values.
// Config file strings are not in here yet!
const (
  ServerPortDefault = 25565

  OpsKey = "OPS"
  // This is how it is usually done.
  // OpsDefault = userName


  ModeKey = "MODE"
  ModeDefault = "creative"

  ViewDistanceKey = "VIEW_DISTANCE"
  ViewDistanceDefault = "10"

  SpawnAnimalsKey = "SPAWN_ANIMALS"
  SpawnAnimalsDefault = "true"

  SpawnMonstersKey = "SPAWN_MONSTERS"
  SpawnMonstersDefault = "false"

  SpawnNPCSKey = "SPAWN_NPCS"
  SpawnNPCSDefault = "true"

  ForceGameModeKey = "FORCE_GAMEMODE"
  ForceGameModeDefault = "true"

  GenerateStructuresKey = "GENERATE_STRUCTURES"
  GenerateStructuresDefault = "true"

  AllowNetherKey = "ALLOW_NETHER"
  AllowNetherDefault = "true"

  MaxPlayersKey = "MAX_PLAYERS"
  MaxPlayersDefault = "20"

  QueryKey = "QUERY"
  QueryDefault = "true"

  QueryPortKey = "QUERY_PORT"
  QueryPortDefault = "25565"

  EnableRconKey = "ENABLE_RCON"
  EnableRconDefault = "true"

  RconPortKey = "RCON_PORT"
  RconPortDefault = "25575"

  RconPasswordKey = "RCON_PASSWORD"
  RconPasswordDefault = "testing"   // TODO: NO NO NO NO NO NO NO NO NO NO

  MOTDKey = "MOTD"
  // This is how it's usually done:
  // MOTDDefault = fmt.Sprintf("A neighborhood kept by %s.", userName)
  PVPKey = "PVP"
  PVPDefault = "false"

  LevelKey = "LEVEL"   // World Save name
  LevelDefault = "world"

  OnlineModeKey = "ONLINE_MODE"
  OnlineModeDefault = "true"

  JVMOptsKey = "JVM_OPTS"
  JVMOptsDefault = "-Xmx1024M -Xms1024M"
)


// Get an existing server from the environment.
func GetServer(clusterName, taskArn string, sess *session.Session) (a *Server, err error){
  dtm, err := awslib.GetDeepTasks(clusterName, sess)
  if err != nil { return a, fmt.Errorf("terminate server failed: %s", err) }
  dt := dtm[taskArn]
  serverEnv, err := dt.GetEnvironment(MinecraftServerContainerName)
  controllerEnv, err := dt.GetEnvironment(MinecraftControllerContainerName)
  userName := serverEnv[ServerUserKey]
  serverName := serverEnv[ServerNameKey]
  serverIp := dt.PublicIpAddress()
  rconPort := serverEnv[RconPortKey]
  rconPW := serverEnv[RconPasswordKey]
  archiveBucket := controllerEnv[ArchiveBucketKey]
  serverDirectory := controllerEnv[ServerLocationKey]
  a = NewServer(userName, serverName, serverIp, 
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

