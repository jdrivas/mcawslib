package mclib

import(
  "testing"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"
)

func TestPublishArchive(t *testing.T) {
  var sess *session.Session = nil
  _, err := PublishArchive("file","buket","path", sess)
  assert.Error(t, err, "PublishAndArchive didn't err on nil session")
}
