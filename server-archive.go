package mclib

import(
  "fmt"
  "sort"
  "time"
)

func (s *Server) TakeServerSnapshot() (*PublishedArchiveResponse, error) {
  files := s.serverSnapshotFiles()
  return s.SafeArchiveThenPublish(files, ServerSnapshot)
}

// func (s *Server) TakeServerSnapshotWithRetry( retries int, waitTime time.Duration) (resp *PublishedArchiveResponse, err error) {
//   return s.SafeArchiveThenPublishWithRetry(retries, waitTime, ServerSnapshot)
// }

func (s *Server) TakeWorldSnapshot() (*PublishedArchiveResponse, error) {
  files := s.worldSnapshotFiles()
  return s.SafeArchiveThenPublish(files, WorldSnapshot)
}

// func (s *Server) TakeWorldSnapshotWithRetry(retries int, waitTime time.Duration) (resp *PublishedArchiveResponse, err error) {
//   return s.SafeArchiveThenPublishWithRetry(retries, waitTime, WorldSnapshot)
// }

func (s *Server) TakeSnapshotWithFiles(files []string) (*PublishedArchiveResponse, error) {
  return s.SafeArchiveThenPublish(files, MiscSnapshot)
}

// Safe archive tries to stop the server from writing to disk using the rcon connection to tell it to
// stop saving. Save is turned back on after the save and the archive that was created is sent off
// to S3 for safe keeping.
func (s *Server) SafeArchiveThenPublish(files []string, aType ArchiveType) ( resp *PublishedArchiveResponse, err error) {
  if s.GoodRcon() && !s.HasRconConnection() {
    _, err = s.NewRcon()
    if err != nil { return resp, fmt.Errorf("Can't create rcon connection for snapshot snapshot: %s", err)}
  }

  resp, err = s.archiveAndPublish(files, aType)
  return resp, err
}


// Convenience wrapper around GetArchives and then pulling the snaps
// from them. See archiveDB.
func (s *Server) ServerSnapshots() (snaps []Archive, err error) {
  snaps, err = GetArchivesForServer(ServerSnapshot, s.User, s.Name, s.ArchiveBucket, s.AWSSession)
  return snaps, err
}

func (s *Server) WorldSnapshots() (snaps []Archive, err error){
  snaps, err = GetArchivesForServer(WorldSnapshot, s.User, s.Name, s.ArchiveBucket, s.AWSSession)
  return snaps, err
}

func (s *Server) LatestWorldSnapshot() (snap *Archive, err error) {
  return s.GetLatestSnapshot(WorldSnapshot)
}

func (s *Server) LatestServerSnapshot() (snap *Archive, err error) {
  return s.GetLatestSnapshot(ServerSnapshot)
}

func (s *Server) GetLatestSnapshot(t ArchiveType) (snap *Archive, err error) {
  snaps, err := GetArchivesForServer(t, s.User, s.Name, s.ArchiveBucket, s.AWSSession)
  if err != nil { return snap, fmt.Errorf("Failed to get the most recent snapshot (%s): %s", t.String(), err) }
  if len(snaps) == 0 {
    err = fmt.Errorf("No %s snapshots found.", t.String())
  } else {
    sort.Sort(ByLastMod(snaps))
    snap = &snaps[len(snaps)-1]
  }
  return snap, err
}

// A Snapshot is a zipfile (archive file) published to the SnapshotPath (archived) on the remote
// service (s3).
func (s *Server) archiveAndPublish(files []string, aType ArchiveType) (resp *PublishedArchiveResponse, err error) {
  path, err  := s.archivePath(aType)
  if err != nil { return nil, err }
  resp, err = ArchiveAndPublish(s.Rcon, files, s.ServerDirectory, s.ArchiveBucket, path, s.AWSSession)
  return resp, err
}

func (s *Server ) archiveFiles(aType ArchiveType) ([]string, error){
  switch aType {
  case ServerSnapshot: return s.serverSnapshotFiles(), nil
  case WorldSnapshot: return s.worldSnapshotFiles(), nil
  }
  return nil, fmt.Errorf("Error with a bad ArchiveType: %s", aType.String())
}

func (s *Server ) archivePath(aType ArchiveType) (string, error){
  return ArchivePath(s.User, s.Name, time.Now(), aType), nil
}

func(s *Server) serverSnapshotFiles() []string {
  files := []string{
    ".",
    // "config",
    // "logs",
    // "mods",
    // "world",
    // "banned-ips.json",
    // "banned-players.json",
    // "server.properties",
    // "usercache.json",
    // "whitelist.json",
  }
  return files
}

// this only backups a vanilla world.
// TODO: Do we try to pick world files based on server type?
// Or do we just backup whatever is there and put all possible worlds here.
func (s *Server) worldSnapshotFiles() []string{
  files := []string{"world"}
  return files
}



