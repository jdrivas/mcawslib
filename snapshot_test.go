package mclib

import (
  // "fmt"
  "testing"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"
  "github.com/Sirupsen/logrus"
)

// We're going to the local configuration file for this.
// Note that in order for it to work, we need a config called "mclib-test"
func testConfig(t *testing.T) (config *aws.Config){
  testProfile := "mclib-test"
  s, err  := session.NewSessionWithOptions(session.Options{
    Profile: testProfile,
    SharedConfigState: session.SharedConfigEnable,
  })
  if assert.NoError(t, err) {
    config = s.Config
  }
  return config
}

func testServer(t *testing.T, useRcon bool) (s *Server) {
  config := testConfig(t)
  if useRcon {
    s = NewServer("testuser", "TestServer", "127.0.0.1", "25575", "testingpw", 
      "craft-config-test", "server", config)
  } else {
    s = NewServer("testuser", "TestServer", "", "0", "", 
      "craft-config-test", "server", config)
  }
  return s
}

func TestNewSnapshotPath(t *testing.T) {
  s := testServer(t, true)
  now := time.Now()  
  timeElement := now.Format(time.RFC3339) + "-"
  serverElement := s.User + "-" + s.Name
  // This should be: user/snapshots/<RFC3339TimeString>-<ServerUser>-<ServerName>-snapshot.zip
  // The snapshotPathELement and snapshotFielExt are private contstants in the library.
  expectedValue := "testuser/" + snapshotPathElement + "/" + timeElement + serverElement + snapshotFileExt
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