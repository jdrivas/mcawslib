package mclib

import (
  "testing"
  "time"
  "github.com/stretchr/testify/assert"
)

func TestNewRcon(t *testing.T) {
  skipOnShort(t)
  s := testServer(t, true)
  assert.NotEqual(t, "", s.ServerIp, "Server IP was not set.")
  assert.NotEqual(t, "", s.RconPort, "RconPort was not set.")
  assert.True(t, s.GoodRcon(), "Server Rcon was not good.")


  rcon, err := s.NewRcon()
  if assert.NoError(t, err, "Server failed NewRcon.") {
    assert.NotNil(t, rcon, "Server.NewRcon returned nil.")
    assert.NotNil(t, s.Rcon, "Server.NewRcon() set Server.Rcon to nil.")
  }
  reply, err := s.Rcon.List()
  assert.NoError(t, err, "Rcon failed List().") 
  assert.Contains(t, reply, "There are", "Failed to get the correct list string back.")
}

func TestNewRconWithRety(t *testing.T) {
  skipOnShort(t)
  s := testServer(t, true)
  assert.NotEqual(t, "", s.ServerIp, "Server IP was not set.")
  assert.NotEqual(t, "", s.RconPort, "RconPort was not set.")
  assert.True(t, s.GoodRcon(), "Server Rcon was not good.")

  rcon, err := s.NewRconWithRetry(1, 1*time.Second)
  if assert.NoError(t, err, "Server failed NewRcon.") {
    assert.NotNil(t, rcon, "Server.NewRcon returned nil.")
    assert.NotNil(t, s.Rcon, "Server.NewRcon() set Server.Rcon to nil.")
  }
  reply, err := s.Rcon.List()
  assert.NoError(t, err, "Rcon failed List().") 
  assert.Contains(t, reply, "There are", "Failed to get the correct list string back.")
}