package mclib

import(
  "fmt"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  // "github.com/aws/aws-sdk-go/service/s3"
  // "github.com/Sirupsen/logrus"
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
  AwsConfig *aws.Config
}

func NewServer(userName, serverName, serverIp string, rconPort string, rconPw, archiveBucket, serverDirectory string, config *aws.Config) (s *Server) {
  s = new(Server)
  s.User = userName
  s.Name = serverName
  s.ServerIp = serverIp
  s.RconPort = rconPort
  s.RconPassword = rconPw
  s.ArchiveBucket = archiveBucket
  s.ServerDirectory = serverDirectory
  s.AwsConfig = config
  return s
}

// Will take snapshot of the server and publish it to the S3 bucket.
// snapshots are stored
//         bucket:/<Server.User>/snapshots/<time.Now()_ansi-time-string>-<Server.User>-<Server.Name>-snapshot.zip
// Zip files are used because that's the standard in minecraft land.
// 
// If serverIP or rconPort are not nil, then an rcon connection to 
// the server will be made to issue save-all then save-off before
// the snapshot is taken and save-on after once the snapshot has
// been ade but before it's been published to S3. THIS IS NOT RECOMMENDED FOR PRDOCUTION.
func (s *Server) SnapshotAndPublish() ( resp *PublishedArchiveResponse, err error) {

  var rcon  *Rcon
  if !s.NoRcon() && !s.Rcon.HasConnection() {
    rcon, err = s.NewRcon()
    if err != nil { return resp, fmt.Errorf("Can't create rcon connection for snapshot snapshot: %s", err)}
  }

  resp, err = s.archiveAndPublish(rcon)
  return resp, err
}

// Will keep trying to get the RCON connection, but sleeping for waitTime and then retrying, up to
// retries times.
// Will fail if the server doesn't have serverIp, rconPort, and rconPassword.
func (s *Server) SnapshotAndPublishWithRetry(retries int, waitTime time.Duration) (resp *PublishedArchiveResponse, err error) {

  if !s.NoRcon() { 
    return nil, fmt.Errorf("Invalid rcon connection paramaters: %s:%s ", s.ServerIp, s.RconPort )
  }
  if len(s.RconPassword) == 0 {
    return nil, fmt.Errorf("No rcon password.")
  }

  var rcon *Rcon
  if !s.Rcon.HasConnection() {
    rcon, err = s.NewRconWithRetry(retries, waitTime)
    s.Rcon = rcon
  }

  resp, err = s.archiveAndPublish(rcon)
  return resp, err
}

func (s *Server) archiveAndPublish(rcon *Rcon) (resp *PublishedArchiveResponse, err error) {
  resp, err = ArchiveAndPublish(rcon, s.ServerDirectory, s.ArchiveBucket, s.newSnapshotPath(time.Now()), s.AwsConfig)
  return resp, err
}

func (s *Server) NewRcon() (rcon *Rcon, err error) {
  rcon, err = NewRcon(s.ServerIp, s.RconPortString(), s.RconPassword)  
  if err != nil {
    s.Rcon = rcon
  }
  return rcon, err
}

func (s *Server) NewRconWithRetry(retries int, waitTime time.Duration) (rcon *Rcon, err error) {
  rcon, err = NewRconWithRetry(s.ServerIp, s.RconPortString(), s.RconPassword, retries, waitTime)
  if err != nil {
    s.Rcon = rcon
  }
  return rcon, err
}

func (s *Server) RconPortString() (string) {
  return fmt.Sprintf("%d", s.RconPort)
}

func (s *Server) NoRcon() (bool) {
  // return len(s.ServerIp) == 0 || s.RconPort == 0
  return len(s.ServerIp) == 0 || len(s.RconPort) == 0
}

const (
  snapshotPathElement = "snapshots"
  snapshotFileExt = "-snapshot.zip"
)

func (s *Server)newSnapshotPath(when time.Time) (string) {
  timeString := when.Format(time.RFC3339)
  return s.User + "/" + snapshotPathElement + "/" + timeString + "-" + s.User + "-" + s.Name + snapshotFileExt
}