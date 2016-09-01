package mclib

import (
  "fmt"
  "testing"
  "time"
  "math/rand"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  "github.com/stretchr/testify/assert"
)

func init() {
  // If you want to repeat a test run with the same values over
  // and over again, just remove this ... or you can give it 
  // a value that you keep using over again.
  rand.Seed(time.Now().UnixNano())
}


func skipOnShort(t *testing.T) {
  if testing.Short() { t.SkipNow() }
}

// We're going to the local configuration file for this.
// Note that in order for it to work, we need a config called "mclib-test"

func testSession(t *testing.T) (sess *session.Session) {
 testProfile := "mclib-test"
  s, err  := session.NewSessionWithOptions(session.Options{
    Profile: testProfile,
    SharedConfigState: session.SharedConfigEnable,
  })
  if assert.NoError(t, err){
    sess = s
  }
  return sess
}

func testConfig(t *testing.T) (config *aws.Config){
  s := testSession(t)
  if assert.NotNil(t, s) {
    config = s.Config
  }
  return config
}

func testServer(t *testing.T, useRcon bool) (s *Server) {
  sess := testSession(t)
  if useRcon {
    s = NewServer("testuser", "TestServer", "192.168.99.100", 25565, "25575", "testing", 
      "craft-config-test", "server", sess)
  } else {
    s = NewServer("testuser", "TestServer", "", 0, "0", "", 
      "craft-config-test", "server", sess)
  }
  return s
}

func testS3Object(key string) (o *s3.Object) {
  when := time.Now()
  o = &s3.Object{
    Key: &key,
    LastModified: &when,
  }
  return o
}


var(
  users = []string{ "Nico", "Pilar", "Elena", "Stephanie", "Josh", "Panda", "Daddy", "jdrivas", }
  servers = []string{ "Test-Server", "Production-Server", "Staging-Server","FrodServer", "CrazyServer", "CraftyServer", }
)

func randUserName() (string) {
  // i := rand.Int() % len(users)
  return users[rand.Int() % len(users)]
}

func randServerName() (string) {
  return servers[rand.Int() % len(servers)]
}


func randomUniqueUserNames(numOfUsers int) (names []string){
  // testNum := (numOfUsers / len(users)) + 1
  names = make([]string, numOfUsers)
  nameSet := make(map[string]bool)
  for i := 0; i < numOfUsers; i++ {
    if numOfUsers > len(users) {
      rn := randUserName()
      for nameSet[rn] {
        rn = randUserName()
        for nameSet[rn] {
          userNumber := rand.Int() & (numOfUsers + 1)
          rn = rn + fmt.Sprintf("-%d", userNumber)
        }
      }
      names[i] = rn
      nameSet[rn] = true
    } else {
      rn := randUserName()
      for nameSet[rn] { rn = randUserName() }
      names[i] = rn
    }
  }
  return names
}




func TestRandomUniqueUsers(t *testing.T) {
  count := 100
  nameSet := make(map[string]bool, count)
  names := randomUniqueUserNames(count)
  assert.Len(t, names, count, "Not the right number of names.")
  for i := 1; i < count; i++ {
    _, ok := nameSet[names[i]]
    assert.False(t, ok, fmt.Sprintf("Found a duplicate name: %s on %dth test", names[i],i+1))
    nameSet[names[i]] = true
  }
}



