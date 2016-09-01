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

func (a Archive) S3Key() (k string) {
  if a.S3Object == nil {
    k = "NO-REFERENCE-TO-S3-OBJECT"
  } else {
    k = *a.S3Object.Key
  }
  return k
}

func (a Archive) LastMod() (t time.Time) {
  if a.S3Object != nil {
    t = a.S3Object.LastModified.Local()
  }
  return t
}

func (a Archive) String() (string) {
  key := a.S3Key()
  return a.Type.String() + ":" + a.UserName + ":" + a.ServerName + ":[" + a.Bucket + "]/" + key
}


// Soring Interface
type ByLastMod []Archive
func (a ByLastMod) Len() int { return len(a) }
func (a ByLastMod) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLastMod) Less(i, j int) bool { return a[i].LastMod().Before(a[j].LastMod())}

type ByUser []Archive
func (a ByUser) Len() int { return len(a) }
func (a ByUser) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByUser) Less(i, j int) bool { return a[i].UserName < a[j].UserName}

type ByType []Archive
func (a ByType) Len() int { return len(a) }
func (a ByType) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByType) Less(i, j int) bool { return a[i].Type < a[j].Type}

type ByBucket []Archive
func (a ByBucket) Len() int { return len(a) }
func (a ByBucket) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBucket) Less(i, j int) bool { return a[i].Bucket < a[j].Bucket}


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
// TODO: Move some of this to awslib.

const (
  snapshotPathElement = "snapshots"
  snapshotFileExt = "-snapshot.zip"
  worldPathElement = "worlds"
  worldFileExt = "-world.zip"
)

var typeToPathElement = map[ArchiveType]string{
  ServerSnapshot: snapshotPathElement,
  World: worldPathElement,
}

var typeToFileExt = map[ArchiveType]string{
  ServerSnapshot: snapshotFileExt,
  World: worldFileExt,
}


// TODO: Move PathJoin to awslib
const S3Delim = "/"
func S3PathJoin(elems ...string ) string {
  s := ""
  for _, e := range elems {
    e = strings.TrimRight(e, S3Delim) // lose all trailing delims
    s += e + S3Delim
  }
  return s
}

const S3BaseURI = "https://s3.amazonaws.com"
// Full qualified URI for a snapshot given the arguments.

func SnapshotURI(bucket, userName, serverName, snapshotFileName string) (string) {
  path := archivePath(userName, serverName, ServerSnapshot)
  return S3PathJoin(S3BaseURI, bucket, path) + snapshotFileName
}


func NewSnapshotPath(userName, serverName string, when time.Time) (string) {
  return newArchivePath(userName, serverName, when, ServerSnapshot)
}

func newArchivePath(userName, serverName string, when time.Time, aType ArchiveType) (string) {
  pathName := archivePath(userName, serverName, aType)
  archiveName := archiveFileName(userName, serverName, when, aType)
  fullPath := pathName + archiveName
  return fullPath
}


// Make the path for an archive based on the constituate values.
// This is the mapping definition (see archiveDb_test)
// <user>/<server>/<archive-type-string>
func archivePath(user, server string, aType ArchiveType) (string) {
  s := S3PathJoin(user, server, typeToPathElement[aType])
  return s
}


// the files names are: <time>-<user>-<server>-<archiveExt>
func archiveFileName(user, server string, when time.Time, aType ArchiveType) (string) {
  timeString := when.Format(time.RFC3339)
  return timeString + "-" + user + "-" + server + typeToFileExt[aType]
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
  // TODO: move this to awslib.
  // GetObjectList(string bucketName, preFix) ([]*s3.Object)
  // You might consider returning a map of ([string]*s3.Object), keyed on  the storage key.
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

