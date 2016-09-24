package mclib

import(
  "bytes"
  "fmt"
  "io"
  "net/http"
  "os"
  "strings"
  "time"
  "archive/zip"
  "path/filepath"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  "github.com/Sirupsen/logrus"
)

type PublishedArchiveResponse struct {
  ArchiveFilename string
  BucketName string
  StoredPath string
  UserName string
  PutObjectOutput *s3.PutObjectOutput
}


// Create a zipfile (archive) at the serverDirectory and publish it to publishPath on S3.
func ArchiveAndPublish(rcon *Rcon, fileNames []string, serverDirectory, bucketName, publishPath string, sess *session.Session) (resp *PublishedArchiveResponse, err error) {
  archiveDir := os.TempDir()
  archiveFileName := fmt.Sprintf("archive-%s.zip", time.Now())
  archivePath := filepath.Join(archiveDir, archiveFileName)

  log.Info(logrus.Fields{"archiveDir": serverDirectory, "bucket": bucketName, "publishPath": publishPath}, "Creating Archive.")

  err = ArchiveServer(rcon, fileNames, serverDirectory, archivePath)
  if err != nil { return nil, err }
  resp, err = PublishArchive(archivePath, bucketName, publishPath, sess)
  return resp, err
}

// Produce an archive of a server.
// Use an rcon connection to first save-all, then save-off before the archive.
// When the archive is finished use rcon to save-on.
// If rcon is nil, then don't do the save-off/save-all/save-on (not for production).
func ArchiveServer(rcon *Rcon, fileNames []string, serverDirectory string, archiveFileName string) (err error) {

  if rcon != nil {
    _, err = rcon.SaveAll()
    if err != nil { return err }
    _, err = rcon.SaveOff()
    if err != nil { return err }
  }

  err = CreateServerArchive(fileNames, serverDirectory, archiveFileName)

  // Make sure this happens no matter what.
  if rcon != nil {
    _,rcErr := rcon.SaveOn()
    if err != nil { return err }
    if rcErr != nil {
      err = fmt.Errorf("ArchiveServer: server archived, problem turning auto-save back on: %s", err)
    }
  }

  log.Info(logrus.Fields{"dir": serverDirectory, "archive": archiveFileName},"Archived server.")
  return err
}

// Make a zipfile of the server directory in directoryName.
// TODO: This currently fails in the face of missing files or directories and
// the archive is not created. We should revist this and dtermine if there is 
// a better or different way.
func CreateServerArchive(fileNames []string, directoryName, zipfileName string) (err error) {

  log.Debug(logrus.Fields{"dir": directoryName, "archive": zipfileName,}, "Archiving server.")
  zipFile, err := os.Create(zipfileName)
  if err != nil { return fmt.Errorf("CreateArchiveServer: can't open zipfile %s: %s", zipfileName, err) }
  defer zipFile.Close()
  archive := zip.NewWriter(zipFile)
  defer archive.Close()

  dir, err := os.Open(directoryName)
  if err != nil { return fmt.Errorf("CreateArchiveServer: can't open server directory %s: %s", directoryName, err) }
  dirInfo, err := dir.Stat()
  if err != nil { return fmt.Errorf("CreateArchiveServer: can't stat directory %s: %s", directoryName, err) }
  if !dirInfo.IsDir() { return fmt.Errorf("CreateArchiveServer: server directory %s is not a directory.") }

  currentDir, err := os.Getwd()
  if err != nil { return fmt.Errorf("CreativeArchiveServer: can't get the current directory: %s", err) }
  defer os.Chdir(currentDir)

  err = dir.Chdir()
  if err != nil { return fmt.Errorf("CreativeArchiveServer: can't change to server directory %s: %s", directoryName, err) }

  log.Debug(logrus.Fields{"length": len(fileNames),}, "Saving files to archive")
  for _, fileName := range fileNames {
    err = writeFileToZip("", fileName, archive)
    // err = writeFileToZip(directoryName, fileName, archive)
    if err != nil {return fmt.Errorf("ArchiveServer: can't write file \"%s\" to archive: %s", fileName, err)}
  }
  return err
}

// This currently errors on unfound files and directories.
// TODO: consider if we want to allow for missings files to be noted but not 
//shutdown everything.  
func writeFileToZip(baseDir, fileName string, archive *zip.Writer) (err error) {

  err = filepath.Walk(fileName, func(path string, info os.FileInfo, err error) (error) {
    if err != nil { return err }

    header, err := zip.FileInfoHeader(info)
    if err != nil { return fmt.Errorf("Couldn't convert FileInfo into zip header: %s", err) }

    if baseDir != "" {
      header.Name = filepath.Join(baseDir, path)
    } else {
      header.Name = path
    }

    if info.IsDir() {
      header.Name += "/"
    } else {
      header.Method = zip.Deflate // Is this necessary?
    }

    log.Debug(logrus.Fields{"zip-header": header.Name,}, "Writing Zip Header.")
    writer, err := archive.CreateHeader(header)
    if err != nil { return fmt.Errorf("Couldn't write header to archive: %s", err)}

    if !info.IsDir() {
        log.Debug(logrus.Fields{"file": path,}, "Opening and copying file to archive")
        file, err := os.Open(path)
        if err != nil { fmt.Errorf("Couldn't open file %s: %s", path, err) }
        _, err = io.Copy(writer, file)
        if err != nil { return fmt.Errorf("io.copy failed: %s", path, err)}
    }

    return err
  })
  return err
}

// Puts the archive in the provided bucket:path on S3 in a 'directory' for the user. Bucket must already exist.
// Config must have keys and region.
func PublishArchive(archiveFileName string, bucketName string, path string, sess *session.Session) (*PublishedArchiveResponse, error) {
  // TODO: it may be good to remove this, but as we were having trouble with
  // sessions for a minute there, the resulting panic wasn't helping, though it's
  // probably the right behavior.
  if sess == nil {return nil, fmt.Errorf("PublishArchive can't have a nil session.Session")}
  s3svc := s3.New(sess)
  file, err := os.Open(archiveFileName)
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't open archive file: %s", err)}
  defer file.Close()

  fileInfo, err := file.Stat()
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't stat archive file: %s: %s", archiveFileName, err)}
  fileSize := fileInfo.Size()

  buffer := make([]byte, fileSize)
  fileType := http.DetectContentType(buffer)
  _, err = file.Read(buffer)
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't read archive file: %s: %s", archiveFileName, err)}
  fileBytes := bytes.NewReader(buffer)

  log.Debug(logrus.Fields{
    "archiveFile": archiveFileName, 
    "bytes": fileSize, 
    "fileType": fileType, 
    "bucket": bucketName, 
    "archive": path}, "PublishArchive: Writing Archive.")

  // TODO: Lookinto this and in particular figure out how to use an iamrole for this.
  aclString := "public-read"

  // Gratuitous use of aws because of some
  // issues with the compiler.
  b := aws.String(bucketName)

  params := &s3.PutObjectInput{
    Bucket: b,
    Key: aws.String(path),
    ACL: aws.String(aclString),
    Body: fileBytes,
    ContentLength: aws.Int64(fileSize),
    ContentType: aws.String(fileType),
  }
  resp, err := s3svc.PutObject(params)

  returnResp := &PublishedArchiveResponse{
    ArchiveFilename: archiveFileName,
    BucketName: bucketName,
    StoredPath: path,
    // UserName: user,
    PutObjectOutput: resp,
  }

  return returnResp, err
}


// Replace .. and absolute paths in archives.
func sanitizedName(fileName string) string {
  fileName = filepath.ToSlash(fileName)
  fileName = strings.TrimLeft(fileName, "/.")
  return strings.Replace(fileName, "../", "", -1)
}


