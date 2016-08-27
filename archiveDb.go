package mclib

import(
  "fmt"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  "github.com/Sirupsen/logrus"
)




// 
// ArchiveType
//

// I miss Ruby atoms.
const (
  ServerSnapshot ArchiveType = iota
  World ArchiveType = iota
  BadType ArchiveType = iota
)
var AllArchiveTypes = []ArchiveType{ServerSnapshot, World, BadType}
type ArchiveType int

func (a ArchiveType) String() (string) {
  switch a {
  case ServerSnapshot: return "Server-Snapshot"
  case World: return "World"
  default: return "INVALID ArchiveType"
  }
}


//
// Archive
//
type Archive struct {
  UserName string
  ServerName string
  Type ArchiveType
  Bucket string
  S3Object *s3.Object
}


func NewArchive(t ArchiveType, userName, serverName string, bucketName string, object *s3.Object) (a Archive) {
  a.UserName = userName
  a.ServerName = serverName
  a.Type = t
  a.Bucket = bucketName
  a.S3Object = object
  return a
}

func (a Archive) String() (string) {
  key := ""
  if a.S3Object == nil {
    key = "NO-REFERENCE-TO-S3-OBJECT"
  } else {
    key = *a.S3Object.Key
  }
  return a.Type.String() + ":" + a.UserName + ":" + a.ServerName + ":[" + a.Bucket + "]/" + key
}


// 
// ArchiveMap
//

// Keyed on user name.
type ArchiveMap map[string][]Archive

func NewArchiveMap() (ArchiveMap) {
  return make(ArchiveMap)
}

func (am ArchiveMap) String() (s string) {

  for key, value := range am {
    s += fmt.Sprintf("** %s - %s ", key, value)
  }
  return s
}

func (am ArchiveMap) Add(a Archive) {
  alist, ok := am[a.UserName] 
  if ok {
    am[a.UserName] = append(alist, a)
  } else {
    am[a.UserName] = []Archive{a}
  }
}

func (am ArchiveMap) GetSnapshots(userName string) (snaps []Archive) {
  return am.GetArchives(userName, ServerSnapshot)
}

func (am ArchiveMap) GetArchives(userName string, t ArchiveType) (archiveList []Archive) {
  archives := am[userName]
  // this is based on the notion that most of the archives will be snapshots.
  if t == ServerSnapshot {
    archiveList = make([]Archive,0, len(archives))
  } else {
    archiveList = make([]Archive, 0, 50)
  }
  for _, archive := range archives {
    if archive.Type == t {
      archiveList = append(archiveList, archive)
    }
  }
  return archiveList
}


//
// Maping to S3
//

const (
  snapshotPathElement = "snapshots"
  snapshotFileExt = "-snapshot.zip"
  worldPathElement = "worlds"
  worldFileExt = "-world.zip"
)

const S3Delim = "/"

// this is the mapping.
func newSnapshotPath(userName, serverName string, when time.Time) (string) {
  timeString := when.Format(time.RFC3339)
  pathName := userName + S3Delim + serverName + S3Delim + snapshotPathElement
  archiveName := timeString + "-" + userName + "-" + serverName + snapshotFileExt

  fullPath := pathName + S3Delim + archiveName
  return fullPath
}

func typeFromS3Key(key *string) (t ArchiveType) {

  keyElems := strings.Split(*key, S3Delim)
  switch keyElems[2] {
  case snapshotPathElement:
    t = ServerSnapshot
  case worldPathElement:
    t = World
  default: 
    t = BadType
  }
  return t
}

func userFromS3Key(key *string) (string) {
  return strings.Split(*key, S3Delim)[0]
}

func archiveFromS3Object(bucketName *string, object *s3.Object) (Archive) {
  // log.Debug(logrus.Fields{"key": *object.Key,},"Archive from object.")
  keyElems := strings.Split(*object.Key, S3Delim)
  a := Archive{
    UserName: keyElems[0],
    ServerName: keyElems[1],
    Type: typeFromS3Key(object.Key),
    Bucket: *bucketName,
    S3Object: object,
  }
  return a
}

func (am ArchiveMap) addFromListResponse(resp *s3.ListObjectsV2Output) {
  // log.Debug(logrus.Fields{"objects": len(resp.Contents)}, "adding objects to a map.")
  for _, object := range resp.Contents {
    archive := archiveFromS3Object(resp.Name, object)
    am.Add(archive)
  }
}

// This is for searching on user.
func getS3ArchivePrefixString(userName string) (string) {
  return userName
}


//
// Functions to get archives from S3.
//

// Blocks until finished.
func GetArchives(userName, bucketName string, session *session.Session) (archives ArchiveMap, err error) {
  s3Svc := s3.New(session)
  archives = NewArchiveMap()
  params := &s3.ListObjectsV2Input{
    Bucket: aws.String(bucketName),
    // Delimiter: aws.String(S3Delim),
    Prefix: aws.String(getS3ArchivePrefixString(userName)),
  }


  isTruncated := true
  for isTruncated {
    resp, err := s3Svc.ListObjectsV2(params)
    if err != nil {
      return archives, 
        fmt.Errorf("Couldn't get objects from bucket %s, with prefix %s: %s", userName, bucketName, err)
    }
    // prefixes := make([]string, len(resp.CommonPrefixes))
    prefixesString := ":"
    for _, p := range resp.CommonPrefixes {
      prefixesString += fmt.Sprintf("%s:",*p.Prefix)
    }
    log.Debug(logrus.Fields{
      "inputPrefix:": *params.Prefix,
      "keyCount": *resp.KeyCount,
      "bucket": *resp.Name,
      "isTruncated": *resp.IsTruncated,
      "noOfCommonPrefixes": len(resp.CommonPrefixes),
      "prefixes": prefixesString,
      "objects": len(resp.Contents),
    }, "Listed a bucket in S3.")

    archives.addFromListResponse(resp)

    // Go get more from the server?
    if *resp.IsTruncated {
      params.ContinuationToken = resp.NextContinuationToken
    } else {
      isTruncated = false
    }
  }

  return archives, err
}

func GetSnapshotList(userName, bucketName string, session *session.Session) (snaps []Archive, err error) {
  am, err := GetArchives(userName, bucketName, session)
  if err == nil {
    snaps = am.GetSnapshots(userName)
  }
  return snaps, err
}

