package v1

var (
	PlantFinalizer = GroupVersion.Group + "/" + string(PlantKind)

	DefaultContainerPort int32 = 80
	DefaultReplicaCount  int32 = 1
)
