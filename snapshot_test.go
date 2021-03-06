package mclib

import (
  // "fmt"
  "testing"
  "time"
  "github.com/stretchr/testify/assert"
  "github.com/Sirupsen/logrus"

  // LOOKING FOR TROUBLE HERE
  // "awslib"
  // "github.com/jdrivas/awslib"
)


func TestNewSnapshotPath(t *testing.T) {
  s := testServer(t, false)
  now := time.Now()  
  timeElement := now.Format(time.RFC3339) + "-"
  serverElement := s.User + "-" + s.Name
  // This should be: <ServerUser>/<ServerName>/snapshots/<RFC3339TimeString>-<ServerUser>-<ServerName>-snapshot.zip
  // The snapshotPathELement and snapshotFielExt are private contstants in the library.
  expectedValue := s.User + "/" + s.Name + "/" + serverPathElement + "/" + timeElement + serverElement + serverFileExt
  testPath, _ := s.archivePath(ServerSnapshot)

  // TODO: I expect that this will fail sometimes due to the time string
  // being hidden in archivePath(). May want to bring that out.
  assert.Equal(t, expectedValue, testPath)
}


// TODO: This is more of an acceptance test that can be used as part of a more generic release process.
// Mostly because it takes to long. So either figure out some VCR like thing, or move it out of unit tests.
func TestSnapshotAndPublish(t *testing.T) {
  if testing.Short() { t.SkipNow()}
  SetLogLevel(logrus.DebugLevel)
  s := testServer(t, false)
  resp, err := s.TakeServerSnapshot()
  if assert.NoError(t, err) {
    // Not much of a test this.
    // This should instead do a regex to ensure that
    // The value returned by newSnapshotPath and this agree
    // The trick is dealing with time, which won't be an eact match.
    // But to be fair, that pretty much turns out to be a test
    // of newSnapshot path because the S3 response on put doesn't
    // actually return the path that it stored the item at.
    assert.Equal(t, s.ArchiveBucket, resp.BucketName)
  }

}

func TestGetSnapshotList(t *testing.T) {
  skipOnShort(t)
  log.SetLevel(logrus.DebugLevel)
  s := testServer(t, false)
  snaps, err := GetArchivesForServer(ServerSnapshot, s.User, s.Name, s.ArchiveBucket, s.AWSSession)
  if assert.NoError(t, err) {
    for i, snap := range snaps {
      key := snap.S3Object.Key
      assert.Equal(t, s.User, userFromS3Key(key), 
        "On try: %d %s, User name was different.", i+1, *key)
      assert.Equal(t, ServerSnapshot, typeFromS3Key(key), 
        "On try: %d %s, Type was not ServerSnapshot - was: %s", i+1, *key, typeFromS3Key(key))
    }
  }
}