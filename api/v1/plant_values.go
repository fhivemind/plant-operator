package v1

const (
	DefaultContainerPort int32 = 80 // DefaultContainerPort defines the default value of ContainerPort for CRD
	DefaultReplicaCount  int32 = 1  // DefaultReplicaCount defines the default value of Replicas for CRD
)

var (
	GroupName = GroupVersion.Group // GroupName exports defined operator group
	Finalizer = GroupName          // Finalizer defines CRD resource finalizer name

	ManagedByLabel = GroupName + "/" + "managed-by" // ManagedByLabel defines a kind-based owner label
	OwnerNameLabel = GroupName + "/" + "owner-name" // OwnerNameLabel defines a resource-based owner label

	OperatorPlant = "plant-operator" // OperatorPlant exports Plant operator name
)

func (plant *Plant) OperatorLabels() map[string]string {
	labels := make(map[string]string)
	labels[ManagedByLabel] = OperatorPlant
	labels[OwnerNameLabel] = plant.Name
	return labels
}
