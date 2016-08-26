package mclib

import (
  "fmt"
  "time"
  "testing"
  "math/rand"
  // "github.com/aws/aws-sdk-go/service/s3"
  "github.com/stretchr/testify/assert"
)

func newArchiveMap(size, noOfUsers int) {
  // am := new(ArchiveMap)
  for i := 0; i < size; i++ {

  }
}

func TestArchiveStringNoObject (t *testing.T) {
  s := testServer(t, false)
  a := NewArchive(ServerSnapshot, s.User, s.Name, s.ArchiveBucket, nil)
  assert.Contains(t, a.String(), s.User)
  assert.Contains(t, a.String(), "NO-REFERENCE-TO-S3-OBJECT")
}

func TestArchiveStringWithObject (t *testing.T) {
  s := testServer(t, false)
  path := s.newSnapshotPath(time.Now())
  o := testS3Object(path)
  a := NewArchive(ServerSnapshot, s.User, s.Name, s.ArchiveBucket, o)
  assert.Contains(t, a.String(), path)
  assert.NotContains(t, a.String(), "NO-REFERENCE-TO-S3-OBJECT")
  assert.NotNil(t, a.S3Object)
}

func getRandomArchiveType() (ArchiveType) {
  return AllArchiveTypes[rand.Int() % len(AllArchiveTypes)]
}

func TestGetSnapshots (t *testing.T) {
  // Config
  iters := 100
  numUniqueUsers := 10

  snapCount := 0
  userSet := make(map[string]bool)
  users := randomUniqueUserNames(numUniqueUsers)
  user := users[rand.Int() % numUniqueUsers]
  sess := testSession(t)
  archiveMap := make(ArchiveMap, iters)
  for i := 1; i < iters; i++ {
    un := users[rand.Int() % len(users)]
    _, ok := userSet[un]
    if !ok {userSet[un] = true} // track how many unique users we added to the map.
    sn := fmt.Sprintf("%s-TestServer",un )
    s := NewServer(un, sn, "127.0.0.1", "25575", "secret", "test-bucket", "server", sess)
    t := getRandomArchiveType()
    if t == ServerSnapshot && un == user {snapCount++}
    path := s.newSnapshotPath(time.Now())

    a := NewArchive(t, un, sn, "test-bucket", testS3Object(path))
    archiveMap.Add(a)
  }
  assert.Len(t, archiveMap, len(userSet), "ArchiveMap didn't add enough archives.")
  snaps := archiveMap.GetSnapshots(user)
  assert.Len(t, snaps, snapCount, "Didn't get the right number of snapshots back.")
}

