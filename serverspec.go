package mclib

import(
  "fmt"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"
  "github.com/Sirupsen/logrus"

  // "awslib"
  "github.com/jdrivas/awslib"
)

// TODO: IMPORTANT
// We're propogating a sort of phase-mismatch between constructing a server
// and reading the details out of the network. 
// Essentially we need to stop relying on ConatinerNames for identifying 
// which container is a server and which contriner is a role and rely on
// The RoleKey environment variable.
// There is an alternative which may make more traditional docker-sense and
// that is the DockerLabels key/value pair set. The trouble is that they are 
// stashed only in a ContainerDefinition (which lives in the TaskDefinition.)
// And so not on a running container, which means we'd still have to use
// container names (define a label "ControllerContainerName", with the value of
// the controller conatiner name in it).
// For now, a TaskDefinition must have a role defined see server.go for the possibilities.
// See (s. ServerSpec) ContainerEnvironmentMap() below for the implementaiton.
type ContainerEnv map[string]string
type ServerTaskEnv map[string]ContainerEnv

type ServerSpec struct {
  TaskDefinition *ecs.TaskDefinition
  Cluster string
  ServerTaskEnv ServerTaskEnv
  AWSSession *session.Session
  ServerConatinerName string
  ControllerContainerName string
}

const(
  CraftServerContainerKey = "CRAFT-SERVER_CONTAINER"
  CraftControllerContainerKey = "CRAFT-CONTROLLER-CONTAINER"
)

// TODO: Validate TaskDefinition for Roles before we try to run it.
// TODO: Consider adding the TD register function here, or providing
// some kind of plug-in to ecs-pilot to validate TDs.
// TODO: validate new server names (upperlowercase, no punct but - and _)??
func NewServerSpec(userName, serverName, region, bucketName, cluster, tdArn string, 
  sess *session.Session) (ss ServerSpec, err error) {

  td, err := awslib.GetTaskDefinition(tdArn, sess)
  if err != nil { return ss, err }
  ste := make(ServerTaskEnv,2)
  ste[CraftServerContainerKey] = DefaultProxiedServerTaskEnv(userName, serverName, cluster, region)
  ste[CraftControllerContainerKey] = DefaultControllerTaskenv(userName, serverName, cluster, region, bucketName)
  ss = ServerSpec{
    TaskDefinition: td,
    Cluster: cluster,
    ServerTaskEnv: ste,
    AWSSession: sess,
  }
  return ss, err
}

func (ss *ServerSpec) defaultLogFields() (logrus.Fields) {
  return logrus.Fields{
    "cluster": ss.Cluster, "TaskDefinition": *ss.TaskDefinition.TaskDefinitionArn,
    "userName": ss.UserName(), "serverName": ss.ServerName(),
    "archiveRegion": ss.ArchiveRegion(), "bucketName": ss.BucketName(),
  }
}

// If returned, the error will be a TaskError
func (ss* ServerSpec) LaunchServer() (s *Server, err error) {

  f := ss.defaultLogFields()

  controlEnv := ss.ControllerContainerEnv()
  if controlEnv[ArchiveBucketKey] == "" {
    controlEnv[ArchiveBucketKey] = MinecraftServerDefaultArchiveBucket
  }

  env, err := ss.ContainerEnvironmentMap()
  if err != nil { return s, NewEmptyTaskError(err.Error()) }

  // Launch the task.
  resp, err := awslib.RunTaskWithEnv(ss.Cluster, *ss.TaskDefinition.TaskDefinitionArn, env, ss.AWSSession)
  if err != nil { 
    log.Error(f, "Failed to launch server from ServerSpec.", err)
    mesg := fmt.Sprintf("Error launching server: %s", err)
    return s, NewEmptyTaskError(mesg)
  }
  taskArn := *resp.Tasks[0].TaskArn


  if len(resp.Tasks) > 1 || len(resp.Failures) > 0 { 
    var mesg string
    switch {
    case len(resp.Tasks) > 1 && len(resp.Failures) > 0:
      mesg = fmt.Sprintf("More than one Task (expected 1) (%d) and Failures returned (%d)",
        len(resp.Tasks), len(resp.Failures))
    case len(resp.Tasks) > 1:
      mesg = fmt.Sprintf("More than one Task (expected 1) (%d)", len(resp.Tasks))
    case len(resp.Failures) > 0:
      mesg = fmt.Sprintf("Failures returned (%d)", len(resp.Failures))
    }
    mesg = fmt.Sprintf("%s. However, server task has been launched.", mesg)
    err = NewTaskError(mesg, resp.Tasks, resp.Failures)  
    f["noOfTasks"] = len(resp.Tasks)
    f["noOfFailures"] = len(resp.Failures)
    log.Error(f, "Error creating server task.", err)
    return s, err
  }

  s, err = GetServer(ss.Cluster, taskArn, ss.AWSSession)
  f["serverType"] = s.CraftType()
  f["taskArn"] = taskArn
  log.Info(f, "Launched server.")
  return s, err
}

func (ss *ServerSpec) ServerContainerEnv() (ContainerEnv) {
  return ss.ServerTaskEnv[CraftServerContainerKey]
}

func (ss *ServerSpec) ControllerContainerEnv() (ContainerEnv) {
  return ss.ServerTaskEnv[CraftServerContainerKey]
}

func (ss *ServerSpec) UserName() (string) {
  return ss.ServerContainerEnv()[ServerUserKey]
}

func (ss *ServerSpec) ServerName() (string) {
  return ss.ServerContainerEnv()[ServerNameKey]
}

func (ss *ServerSpec) ArchiveRegion() (string) {
  return ss.ControllerContainerEnv()[ArchiveRegionKey]
}

func (ss *ServerSpec) BucketName() (string) {
  return ss.ControllerContainerEnv()[ArchiveBucketKey]
}

// This takes the environments that we have been indexing on "role" (be careful this isn't 
// consistent yet), and turns them into the awslib map indexed on Container.Name
// We look for the role Environment variable in the ContainerDefinition 
// and take the container name from that ContainerDefinition.
// Which we then use build out the CEM from specs environment.
func (s *ServerSpec) ContainerEnvironmentMap() (cem awslib.ContainerEnvironmentMap, err error) {

  td := s.TaskDefinition
  serverContainerName, ok  := ContainerNameForRole(CraftServerRole, td)
  if !ok { 
    err = fmt.Errorf("Error finding container name for server role in TaskDefinition: %s", *td.TaskDefinitionArn)
    return cem, err
  }
  controllerContainerName, ok  := ContainerNameForRole(CraftControllerRole, td)
  if !ok { 
    err = fmt.Errorf("Error finding container name for container role in TaskDefinition: %s", *td.TaskDefinitionArn)
    return cem, err
  }

  cem = make(awslib.ContainerEnvironmentMap, 2)
  cem[serverContainerName] = s.ServerTaskEnv[CraftServerContainerKey]
  cem[controllerContainerName] = s.ServerTaskEnv[CraftControllerContainerKey]
  return cem, err
}

// Search through the TaskDefinitions ConatinerDefinitions for the env[RoleKey] == r
func ContainerNameForRole(r string, td *ecs.TaskDefinition) (n string, ok bool) {
  for _, cd := range td.ContainerDefinitions {
    for _, kvp := range cd.Environment {
      if *kvp.Name == RoleKey {
        if *kvp.Value == r {
          n = *cd.Name
          ok = true
          break
        }
      }
    }
  }
  return n, ok
}


func DefaultProxiedServerTaskEnv(userName, serverName, cluster, region string) ContainerEnv {
  cenv := DefaultServerTaskEnv(userName, serverName, cluster, region)
  cenv[OnlineModeKey] = ProxiedServerOnlineModeDefault
  return cenv
}

// Region not taken from SESS to enable delployments from other regions.
func DefaultServerTaskEnv(userName, serverName , cluster, region string) ContainerEnv {
  cenv := ContainerEnv{
    RoleKey: CraftServerRole,
    ServerUserKey: userName,
    ServerNameKey: serverName,
    ClusterNameKey: cluster,
    OpsKey: userName,
    // "WHITELIST": "",
    ModeKey: ModeDefault,
    ViewDistanceKey: ViewDistanceDefault,
    SpawnAnimalsKey: SpawnAnimalsDefault,
    SpawnMonstersKey: SpawnMonstersDefault,
    SpawnNPCSKey: SpawnNPCSDefault,
    ForceGameModeKey: ForceGameModeDefault,
    GenerateStructuresKey: GenerateStructuresDefault,
    AllowNetherKey: AllowNetherDefault,
    MaxPlayersKey: MaxPlayersDefault,
    QueryKey: QueryDefault,
    QueryPortKey: QueryPortDefaultString,
    EnableRconKey: EnableRconDefault,
    RconPortKey: RconPortDefaultString,
    RconPasswordKey: RconPasswordDefault, // TODO NO NO NO NO NO NO NO NO NO NO NO NO NO
    MOTDKey: fmt.Sprintf("A neighborhood kept by %s.", userName),
    PVPKey: PVPDefault,
    LevelKey: LevelDefault,
    OnlineModeKey: OnlineModeDefault,
    JVMOptsKey: JVMOptsDefault,
    "AWS_REGION": region,
  }
  return cenv
}

func DefaultControllerTaskenv(userName, serverName, cluster, region, bucketName string) ContainerEnv {
  // Set AWS_REGION to pass the region automatically
  // to the minecraft-controller. The AWS-SDK looks for this
  // env when setting up a session (this also plays well with
  // using IAM Roles for credentials).
  // TODO: Consider moving each of these envs into their own
  // separate basic defaults, which can be leveraged into
  // the separate proxy and barse verions.
  // DRY
  cenv := ContainerEnv{
    RoleKey: CraftControllerRole,
    ServerUserKey: userName,
    ServerNameKey: serverName,
    ArchiveRegionKey: region,
    ArchiveBucketKey: bucketName,
    ServerLocationKey: ServerLocationDefault,
    ClusterNameKey: cluster,
    "AWS_REGION": region,
  }  
  return cenv
}
