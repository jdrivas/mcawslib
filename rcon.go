package mclib

import(
  "fmt"
  "regexp"
  "strconv"
  "time"
  "github.com/bearbin/mcgorcon"
  "github.com/Sirupsen/logrus"
  )

type Rcon struct {
  Host string
  Port int
  Password string
  Client *mcgorcon.Client
}


// create a new connection.
func NewRcon(host string, port string, pw string) (rcon *Rcon, err error) {
  p, err := strconv.Atoi(port)
  if err == nil {
    rcon = &Rcon{
      Host: host, 
      Port: p,
      Password: pw,
    }
    var client mcgorcon.Client
    client, err = mcgorcon.Dial(rcon.Host, rcon.Port, rcon.Password)
    if err == nil {
      log.Debug(logrus.Fields{"server": rcon.Host, "port": rcon.Port,},"NewRcon: connected to server")
      rcon.Client = &client
    }
  }
  return  rcon, err
}

// Create a new connection, retry retries times, waiting retryWait between. 
// This blocks until either a connection is made or until we've done this retryCount times.
// This waits retryWait each time, no exponential back-off.
func NewRconWithRetry(host, port, pw string, retries int, retryWait time.Duration ) (rcon *Rcon, err error) {

  f := logrus.Fields{
    "connection": host + ":" + port, 
    "wait": retryWait, 
    "count": 0,
  }

  retryCount := 0
  for rcon == nil {
    retryCount++
    f["count"] = retryCount
    rcon, err = NewRcon(host, port, pw)
    if err != nil {
      log.Info(f, "RCON Connection failed. Retrying.")
      rcon = nil
    }
    if retryCount > retries { break }
    time.Sleep(retryWait)
  }
  if  rcon == nil {
    log.Info(f, "RCON Failed to create an RCON to the server.")
  } else {
    log.Info(f, "RCON Connected to server.")
  }

  return rcon, err
}

func (rc *Rcon) HasConnection() bool {
  return rc.Client != nil
}

func (rc *Rcon) Send(command string) (reply string, err error ) {
  if rc.Client == nil { return reply, fmt.Errorf("Rcon: Host connection empty.")}
  reply, err = rc.Client.SendCommand(command)
  if err != nil { err = fmt.Errorf("Failed to send \"%s\" to server: %s", command, err)}
  return reply, err
}

func (rc *Rcon) SaveOn() (reply string, err error){
  return rc.Send("save-on")
}

func (rc *Rcon) SaveOff() (reply string, err error) {
  return rc.Send("save-off")
}

func (rc *Rcon) SaveAll() (reply string, err error) {
  return rc.Send("save-all")
}

func (rc *Rcon)List() (reply string, err error) {
  return rc.Send("list")
}

func (rc *Rcon)NumberOfUsers() (nu int, err error) {
  exp := "There are (\\d+)/\\d+.*"
  re := regexp.MustCompile(exp)
  reply, err := rc.List()
  if err == nil {
    matches := re.FindStringSubmatch(reply)
    if len(matches) < 2 {
      err = fmt.Errorf("Rcon: couldn't find number of users. Regex: \"%s\" Reply: \"%s\"", exp, reply)
    } else {
      nu, err = strconv.Atoi(matches[1])
      if err != nil { 
        err = fmt.Errorf("Rcon: failed to get users from Regex: \"%s\" Reply: \"%s\", Match: \"%s\"", 
          exp, reply, matches[1]) 
      }
    }
  }
  return nu, err
}

