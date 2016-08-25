package mclib

import (
  "fmt"
  "os"
  "github.com/go-ini/ini"
  "github.com/Sirupsen/logrus"
)

func init() {
  ini.PrettyFormat = false
}
type ServerConfig struct {
  Config *ini.File
}

func defaultSection(config *ini.File) (*ini.Section) {
  section, err := config.GetSection("")
  log.CheckFatalError(nil, "Couldn't get the default config Section", err)
  return section
}

func  NewConfigFromFile(fileName string) (*ServerConfig) {
  cfg, err := ini.Load(fileName)
  log.CheckFatalError(logrus.Fields{"config": fileName,}, "Can't load config.", err)
  return &ServerConfig{Config: cfg}
}

func (cfg *ServerConfig) WriteToFile(filename string) {
  file, err := os.Create(filename)
  defer file.Close()

  log.CheckFatalError(logrus.Fields{"config": filename}, "Can't open new config file.", err)
  _, err = cfg.Config.WriteTo(file)
  log.CheckFatalError(logrus.Fields{"config": filename}, "Can't write to config file.", err)
}

func (cfg *ServerConfig) SetEntry(key string, value string) {
  f := logrus.Fields{"key": key, "value": value,}
  section  := defaultSection(cfg.Config)
  if section.HasKey(key) {
    entry, err := section.GetKey(key)
    log.CheckFatalError(f, "HasKey() == true but GetKey failed.", err)
    entry.SetValue(value)
  } else {
    log.Info(f, "Key not prsent in config. Configuration unmodified.")
  }
}

func (cfg *ServerConfig) HasKey(key string) (bool) {
  section := defaultSection(cfg.Config)
  return section.HasKey(key)
}

func (cfg *ServerConfig) List() {
  keyHash := defaultSection(cfg.Config).KeysHash()
  for key, value := range keyHash {
    fmt.Printf("%s: %s\n", key, value)
  }
}


