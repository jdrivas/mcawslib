package mclib

import (
  "testing"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"
)

func skipOnShort(t *testing.T) {
  if testing.Short() { t.SkipNow() }
}

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
    s = NewServer("testuser", "TestServer", "192.168.99.100", "25575", "testing", 
      "craft-config-test", "server", config)
  } else {
    s = NewServer("testuser", "TestServer", "", "0", "", 
      "craft-config-test", "server", config)
  }
  return s
}