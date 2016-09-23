package mclib

import(
  "time"
)

func (s *Server) NewRcon() (rcon *Rcon, err error) {
  rcon, err = NewRcon(s.PublicServerIp, s.RconPort.String(), s.RconPassword)  
  if err == nil {
    s.Rcon = rcon
  }
  return rcon, err
}

// Gets a new Rcon connection for the seever. Will retry after waitTime if the connection attempt fails,
// will try up to retry times. Blocks until finished.
func (s *Server) NewRconWithRetry(retries int, waitTime time.Duration) (rcon *Rcon, err error) {
  rcon, err = NewRconWithRetry(s.PublicServerIp, s.RconPort.String(), s.RconPassword, retries, waitTime)
  if err == nil {
    s.Rcon = rcon
  }
  return rcon, err
}

func (s *Server) HasRconConnection() (bool) {
  if s.Rcon == nil {
    return false
  }
  return s.Rcon.HasConnection()
}

func (s *Server) NoRcon() (bool) {
  return len(s.PublicServerIp) == 0 || s.RconPort == 0
  // return len(s.ServerIp) == 0 || len(s.RconPort) == 0
}

func (s *Server) GoodRcon() (bool) {
  return !s.NoRcon()
}