package mclib

import(
  "archive/zip"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "testing"
  "time"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/stretchr/testify/assert"
)

func TestPublishArchive(t *testing.T) {
  var sess *session.Session = nil
  _, err := PublishArchive("file","bucket","path", sess)
  assert.Error(t, err, "PublishAndArchive didn't err on nil session")
}

  
const TestServerDir = "TestingServer"
func makeTempServerDirName() (string) {
  return filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s", TestServerDir, time.Now().Format(time.RFC3339)))
}

// returns the server directory name as sd.
type DirContents struct {
  DirName string
  Files map[string]bool
  Dirs map[string]bool
}

func createTestServerDirectory() (dc *DirContents, err error) {
  sd := makeTempServerDirName()
  err = os.Mkdir(sd, 0777)
  if err != nil { return dc, err }

  dc = new(DirContents)
  dc.DirName = sd
  dc.Files = make(map[string]bool, 0)
  dc.Dirs = make(map[string]bool, 0)
  dirs := []string{"world", "logs"}
  for _, d := range dirs {
    dc.Dirs[d] = true
    dp := filepath.Join(sd, d)
    err = os.Mkdir(dp, 0777)
    if err != nil { return dc, fmt.Errorf("Failed to create directory (%s): %s", dp, err) }
  }

  files := []string{"world/data", "ops.json", "logs/latest.log", "logs/2016-10-11.1.log.gz"}
  for _, f := range files {
    dc.Files[f] = true
    p := filepath.Join(sd, f)
    fp, err := os.Create(p)
    if err != nil { return dc, fmt.Errorf("Failed to create file (%s): %s", p, err) }

    _, err = fp.WriteString("TESTING CONTENT\n")
    if err != nil { return dc, fmt.Errorf("Failed to write data to file (%s): %s", p, err) }
  }

  return dc, err
}

const TestZipFileName = "server.zip"
func makeTempServerZipFileName() (string) {
  return filepath.Join(os.TempDir(), fmt.Sprintf("%sls_%s", time.Now().Format(time.RFC3339), TestZipFileName))
}

func TestWorldZip(t *testing.T) {

  sd, err := createTestServerDirectory()
  if err == nil {
    // fmt.Printf("Serverdir: %s\n", sd.DirName)
    defer os.RemoveAll(sd.DirName)
  } else {
    assert.FailNow(t, err.Error())
  }

  // Determine what we want to zip up.
  files := []string{"world"}
  for f, _ := range sd.Dirs {
    if f != "world" { 
      sd.Dirs[f] = false 
      // fmt.Printf("Removed: %s\n", f)
    } else {
      // fmt.Printf("Not removing: %s\n", f)
    }
  }
  for f, _ := range sd.Files {
    if !strings.HasPrefix(f, "world") { 
      // fmt.Printf("Removing: %s\n", f)
      sd.Files[f] = false 
    } else {
      // fmt.Printf("Not removing %s\n", f)
    }
  }
  archiveAndTestZip(t, files, sd)
}

func TestFullZip(t *testing.T) {
  sd, err := createTestServerDirectory()
  if err == nil {
    defer os.RemoveAll(sd.DirName)
  } else {
    assert.FailNow(t, err.Error())
  }

  files := []string{"."}
  sd.Dirs["."] = true
  archiveAndTestZip(t, files, sd)
}

func archiveAndTestZip(t *testing.T, files []string, sd *DirContents) {
  zfile := makeTempServerZipFileName()
  err := CreateServerArchive(files, sd.DirName, zfile)
  if err == nil {
    defer os.Remove(zfile)
  }
  assert.NoError(t, err, "Failed to create server archive.")

  // Check for files.
  zr, err  := zip.OpenReader(zfile)
  assert.NoError(t, err, "Failed to open the zipfile")
  defer zr.Close()
  for _, f := range zr.File {
    name := strings.Trim(f.Name,"/")
    if f.FileHeader.FileInfo().IsDir() {
      if !sd.Dirs[name] {assert.Fail(t, fmt.Sprintf("Zipfile has an additional directory: (%s)", name))}
      sd.Dirs[name] = false
    } else {
      if !sd.Files[name] {assert.Fail(t, fmt.Sprintf("Zipfile has an additional file: (%s) "), name)}
      sd.Files[name] = false
    }
  }
  for f, notFound := range sd.Files {
    if notFound {
      assert.Fail(t, fmt.Sprintf("Didn't find an expected file in zip archive: %s", f))
    }
  }
  for f, notFound := range sd.Dirs {
    if notFound { 
      assert.Fail(t, fmt.Sprintf("Didn't find an expected directory in the zip archive: %s", f))
    }
  }
}
