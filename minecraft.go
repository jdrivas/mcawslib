package mclib

import (
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"
)

var(
  // log = logrus.New()
  log = sl.New()
)

func init() {
  configureLogs()
}

func SetLogLevel(l logrus.Level) {
  log.SetLevel(l)
}

func SetLogFormatter(f logrus.Formatter) {
  log.SetFormatter(f)
}

func configureLogs() {
  formatter := new(sl.TextFormatter)
  formatter.FullTimestamp = true
  log.SetFormatter(formatter)
  log.SetLevel(logrus.InfoLevel)
}
