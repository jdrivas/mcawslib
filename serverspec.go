package mclib

import(
  "fmt"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/ecs"

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
func NewServerSpec(userName, serverName, region, bucketName, tdArn string, 
  sess *session.Session) (ss ServerSpec, err error) {

  td, err := awslib.GetTaskDefinition(tdArn, sess)
  if err != nil { return ss, err }
  ste := make(ServerTaskEnv,2)
  ste[CraftServerContainerKey] = DefaultProxiedServerTaskEnv(userName, serverName, region)
  ste[CraftControllerContainerKey] = DefaultControllerTaskenv(userName, serverName, region, bucketName)
  ss = ServerSpec{
    TaskDefinition: td,
    ServerTaskEnv: ste,
    AWSSession: sess,
  }
  return ss, err
}

func (s ServerSpec) ServerContainerEnv() (ContainerEnv) {
  return s.ServerTaskEnv[CraftServerContainerKey]
}

func (s ServerSpec) ControllerContainerEnv() (ContainerEnv) {
  return s.ServerTaskEnv[CraftServerContainerKey]
}

// This takes the environments that we have been indexing on "role" (be careful this isn't 
// consistent yet), and turns them into the awslib map indexed on Container.Name
// We look for the role Environment variable in the ContainerDefinition 
// and take the container name from that ContainerDefinition.
// Which we then use build out the CEM from specs environment.
func (s ServerSpec) ContainerEnvironmentMap() (cem awslib.ContainerEnvironmentMap, err error) {

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


func DefaultProxiedServerTaskEnv(userName, serverName, region string) ContainerEnv {
  cenv := DefaultServerTaskEnv(userName, serverName, region)
  cenv[OnlineModeKey] = ProxiedServerOnlineModeDefault
  return cenv
}

// Region not taken from SESS to enable delployments from other regions.
func DefaultServerTaskEnv(userName, serverName , region string) ContainerEnv {
  cenv := ContainerEnv{
    RoleKey: CraftServerRole,
    ServerUserKey: userName,
    ServerNameKey: serverName,
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

func DefaultControllerTaskenv(userName, serverName, region, bucketName string) ContainerEnv {
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
    "AWS_REGION": region,
  }  
  return cenv
}
