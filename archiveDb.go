package mclib

import(
  "fmt"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  "github.com/Sirupsen/logrus"

  // "awslib"
  "github.com/jdrivas/awslib"
)

// 
// ArchiveType
//

// I miss Ruby atoms.
const (
  ServerSnapshot ArchiveType = iota
  WorldSnapshot
  MiscSnapshot
  BadType
)
var AllArchiveTypes = []ArchiveType{ServerSnapshot, WorldSnapshot, MiscSnapshot, BadType}
type ArchiveType int
var archiveTypeToString = map[ArchiveType]string{
  ServerSnapshot: "ServerSnapshot",
  WorldSnapshot: "WorldSnapshot",
  MiscSnapshot: "MiscSnapshot",
  BadType: "BadType",
}
var archiveStringToType = makeArchiveStringToType()
func makeArchiveStringToType() map[string]ArchiveType {
  m := make(map[string]ArchiveType, len(AllArchiveTypes))
  for _, t := range AllArchiveTypes {
    m[archiveTypeToString[t]] = t
  }
  return m
}

func (a ArchiveType) String() (string) {
  as := "INVALID ArchiveType"
  if s, ok := archiveTypeToString[a]; ok {as = s}
  return as
}

func ArchiveTypeFrom(s string) (ArchiveType) {
  at := BadType
  if t, ok := archiveStringToType[s]; ok { at = t }
  return at
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

// TODO: Remove this for using static const kbants. a := &Arcive{UserName: "me" "ServerName": server, ....  }
func NewArchive(t ArchiveType, userName, serverName string, bucketName string, object *s3.Object) (a Archive) {
  a.UserName = userName
  a.ServerName = serverName
  a.Type = t
  a.Bucket = bucketName
  a.S3Object = object
  return a
}

func (a Archive) URI() (uri string) {
  return awslib.S3URI(a.Bucket, a.S3Key())
  // return fmt.Sprintf("https://s3.amazonaws.com/%s/%s", a.Bucket, a.S3Key())
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
// These are all used: sort.Sort(ByLasMod(myArchiveList))
func ByLastMod(aList []Archive) (archiveSort) {
  return archiveSort{ 
    as: aList, 
    less: func(aI, aJ *Archive) (bool) { return aI.LastMod().Before(aJ.LastMod())},
  }
}

func ByType(aList []Archive) (archiveSort) {
  return archiveSort{
    as: aList,
    less: func(aI, aJ *Archive) bool { return aI.Type.String() < aJ.Type.String() },
  }
}

func ByUser(aList []Archive) (archiveSort) {
  return archiveSort{
    as: aList,
    less: func(aI, aJ *Archive) bool { return aI.UserName < aJ.UserName } ,
  }
}

func ByBucket(aList []Archive) (archiveSort) {
  return archiveSort{
    as: aList,
    less: func(aI, aJ *Archive) bool { return aI.Bucket < aJ.Bucket },
  }
}

type archiveSort struct {
  as []Archive
  less func( aI, aJ *Archive) (bool)
}
func (a archiveSort) Len() int { return len(a.as) }
func (a archiveSort) Swap(i, j int) { a.as[i], a.as[j] = a.as[j], a.as[i] }
func (a archiveSort) Less(i, j int) bool { return a.less( &a.as[i], &a.as[j]) }


// 
// ArchiveMap
//

// A list of archives indexed by 
type ArchiveMap map[string]map[string][]Archive

func NewArchiveMap() (ArchiveMap) {
  return make(ArchiveMap)
}

// Gets all of the archives for a user. Returns them in an ArchiveMap for convenience.
func GetArchives(userName, bucketName string, session *session.Session) (archives ArchiveMap, err error) {

  // TODO: move this to awslib.
  // GetObjectList(string bucketName, preFix) ([]*s3.Object)
  // You might consider returning a map of ([string]*s3.Object), keyed on  the storage key.

  s3Svc := s3.New(session)
  archives = NewArchiveMap()

  // These params get objects for a particular user.
  params := &s3.ListObjectsV2Input{
    Bucket: aws.String(bucketName),
    // Delimiter: aws.String(S3Delim),
    Prefix: aws.String(getS3ArchivePrefixString(userName)),
  }

  // Loop over as many as there are .....
  // TODO:  Move this to S3 and set it up with a proccess function.
  isTruncated := true
  for isTruncated {
    resp, err := s3Svc.ListObjectsV2(params)
    if err != nil {
      nerr := fmt.Errorf("Couldn't get objects from bucket %s, with prefix %s: %s", 
        userName, bucketName, err)
      return archives, nerr
    }
    archives.addFromListResponse(resp)

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

    // Go get more from the server?
    if *resp.IsTruncated {
      params.ContinuationToken = resp.NextContinuationToken
    } else {
      isTruncated = false
    }
  }

  log.Debug(logrus.Fields{"noOfArchives": len(archives[userName])},"Returning archive list.")
  return archives, err
}

// Adds to the archive map with an S3 response from listObjects.
func (am ArchiveMap) addFromListResponse(resp *s3.ListObjectsV2Output) {
  // log.Debug(logrus.Fields{"objects": len(resp.Contents)}, "adding objects to a map.")
  for _, object := range resp.Contents {
    archive := archiveFromS3Object(resp.Name, object)
    am.Add(archive)
  }
}

func (am ArchiveMap) String() (s string) {

  for key, value := range am {
    s += fmt.Sprintf("** %s - %s ", key, value)
  }
  return s
}

func (am ArchiveMap) Add(a Archive) {

  // Get the byServerMap of Archives
  byServerMap, ok := am[a.UserName] 
  if !ok { 
    am[a.UserName] = make(map[string][]Archive)
    byServerMap = am[a.UserName]
  }

  // Get the archive list from it and add the archive.
  aList, ok  := byServerMap[a.ServerName]
  if ok { 
    byServerMap[a.ServerName] = append(aList, a) 
  } else {
    byServerMap[a.ServerName] = []Archive{a}
  }
}


// General filter for archive based on archive type.
func (am ArchiveMap) GetArchives(userName string, t ArchiveType) (archiveList []Archive) {
  byServerMap := am[userName]
  archiveList = make([]Archive,0,0)
  for _, aList := range byServerMap {
    for _, archive := range aList {
      if archive.Type == t {
        archiveList = append(archiveList, archive)
      }
    }
  }
  return archiveList
}

// Filter to just get the list of archives for a server.
func (am ArchiveMap) GetArchivesForServer(userName, serverName string, t ArchiveType) (archiveList []Archive) {
  byServerMap := am[userName]
  al := byServerMap[serverName]
  return al
}

// Convienence to filter on type before you have an Archive Map. (calls GetArchives and appropriate filter.)
func GetArchivesForServer(t ArchiveType, userName, serverName, bucketName string, session *session.Session) (snaps []Archive, err error) {
  am, err := GetArchives(userName, bucketName, session)
  if err == nil {
    snaps = am.GetArchivesForServer(userName, serverName, t)
  }
  return snaps, err
}

//
// Maping to S3
// TODO: Move some of this to awslib.

const (
  serverPathElement = "server"
  serverFileExt = "-server.zip"
  worldPathElement = "worlds"
  worldFileExt = "-world.zip"
  miscPathElement = "misc"
  miscFileExt = "-misc.zip"
)

// TODO: Automate the construction of these two.
var typeToPathElement = map[ArchiveType]string{
  ServerSnapshot: serverPathElement,
  WorldSnapshot: worldPathElement,
  MiscSnapshot: miscPathElement,
}

var typeToFileExt = map[ArchiveType]string{
  ServerSnapshot: serverFileExt,
  WorldSnapshot: worldFileExt,
  MiscSnapshot: miscFileExt,
}

// Helper to get the type
func typeFromS3Key(key *string) (t ArchiveType) {
  keyElems := strings.Split(*key, S3Delim)
  switch keyElems[2] {
  case serverPathElement:
    t = ServerSnapshot
  case worldPathElement:
    t = WorldSnapshot
  case miscPathElement:
    t = MiscSnapshot
  default: 
    t = BadType
  }
  // fmt.Printf("%s: %s => type %s\n", *key, keyElems[2], t.String())
  return t
}

// 
// SPEC for the mapping between archive and place in S3 file system.
//

// Full qualified URI for a ServerSnapshot given the arguments.
func ServerSnapshotURI(bucket, userName, serverName, snapshotFileName string) (string) {
  path := archivePath(userName, serverName, ServerSnapshot)
  return S3PathJoin(S3BaseURI, bucket, path) + snapshotFileName
}

// func NewServerSnapshotPath(userName, serverName string, when time.Time) (string) {
//   return newArchivePath(userName, serverName, when, ServerSnapshot)
// }

func ArchivePath(userName, serverName string, when time.Time, aType ArchiveType) (string) {
  pathName := archivePath(userName, serverName, aType)
  archiveName := archiveFileName(userName, serverName, when, aType)
  fullPath := pathName + archiveName
  return fullPath
}

//
// Helpers to implement the spec that maps user, server-name, type to a particular
// place in the S3 file system.
//

const S3BaseURI = "https://s3.amazonaws.com"

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

// Make the path for an archive based on the constituent values.
// This is the mapping definition (see archiveDb_test)
// <user>/<server>/<archive-type-string>
func archivePath(user, server string, aType ArchiveType) (string) {
  s := S3PathJoin(user, server, typeToPathElement[aType])
  return s
}

// the files names are: <time>-<user>-<server>-<archiveExt>
func archiveFileName(user, server string, when time.Time, aType ArchiveType) (string) {
  if user == "" {
    user = "<no-user>"
  }
  if server == "" {
    server = "<no-server>"
  }
  timeString := when.Format(FormatForTimeName)
  return timeString + "-" + user + "-" + server + typeToFileExt[aType]
}


// helper to get the user.
func userFromS3Key(key *string) (string) {
  return strings.Split(*key, S3Delim)[0]
}



// One archive object from the S3 object.
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
  // fmt.Printf("\nNewObjectFromS3: %s\n", *object.Key)
  // fmt.Printf("%#v\n", a)
  return a
}



// This is for searching on user.
func getS3ArchivePrefixString(userName string) (string) {
  return userName
}


