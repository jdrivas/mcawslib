package mclib

import (
  // "fmt"
  "testing"
  "time"
  "github.com/stretchr/testify/assert"
  "github.com/Sirupsen/logrus"
)


func TestNewSnapshotPath(t *testing.T) {
  s := testServer(t, false)
  now := time.Now()  
  timeElement := now.Format(time.RFC3339) + "-"
  serverElement := s.User + "-" + s.Name
  // This should be: <ServerUser>/<ServerName>/snapshots/<RFC3339TimeString>-<ServerUser>-<ServerName>-snapshot.zip
  // The snapshotPathELement and snapshotFielExt are private contstants in the library.
  expectedValue := s.User + "/" + s.Name + "/" + snapshotPathElement + "/" + timeElement + serverElement + snapshotFileExt
  testPath := s.newSnapshotPath(now)

  assert.Equal(t, expectedValue, testPath)
}


// TODO: This is more of an acceptance test that can be used as part of a more generic release process.
// Mostly because it takes to long. So either figure out some VCR like thing, or move it out of unit tests.
func TestSnapshotAndPublish(t *testing.T) {
  if testing.Short() { t.SkipNow()}
  SetLogLevel(logrus.DebugLevel)
  s := testServer(t, false)
  _, err := s.SnapshotAndPublish()
  if assert.NoError(t, err) {
  }

}