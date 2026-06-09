package internal

type ResourceType string

const (
	ResourceTypeCluster   ResourceType = "cluster"
	ResourceTypeNodegroup ResourceType = "nodegroup"
	ResourceTypeAddon     ResourceType = "addon"
)

type ResourceMetadata struct {
	Type             ResourceType
	ComponentID      string
	ComponentTitle   string
	ComponentType    string
	ComponentDesc    string
	ComponentPurpose string
	InventoryType    string
	LabelPrefix      string
}

func GetResourceMetadata(resourceType ResourceType) ResourceMetadata {
	switch resourceType {
	case ResourceTypeCluster:
		return ResourceMetadata{
			Type:             ResourceTypeCluster,
			ComponentID:      "common-components/amazon-eks",
			ComponentTitle:   "Amazon EKS",
			ComponentType:    "service",
			ComponentDesc:    "Amazon Elastic Kubernetes Service provides managed Kubernetes control planes in AWS.",
			ComponentPurpose: "To provide managed Kubernetes cluster control planes for containerized workloads.",
			InventoryType:    "container-orchestration",
			LabelPrefix:      "aws-eks-cluster",
		}
	case ResourceTypeNodegroup:
		return ResourceMetadata{
			Type:             ResourceTypeNodegroup,
			ComponentID:      "common-components/amazon-eks-managed-node-group",
			ComponentTitle:   "Amazon EKS Managed Node Group",
			ComponentType:    "service",
			ComponentDesc:    "Amazon EKS managed node groups provide managed EC2 worker node lifecycle integration for EKS clusters.",
			ComponentPurpose: "To provide managed compute capacity for Kubernetes workloads running on Amazon EKS.",
			InventoryType:    "compute",
			LabelPrefix:      "aws-eks-nodegroup",
		}
	case ResourceTypeAddon:
		return ResourceMetadata{
			Type:             ResourceTypeAddon,
			ComponentID:      "common-components/amazon-eks-addon",
			ComponentTitle:   "Amazon EKS Add-on",
			ComponentType:    "service",
			ComponentDesc:    "Amazon EKS add-ons provide managed operational software components for EKS clusters.",
			ComponentPurpose: "To manage lifecycle and health of EKS-supported cluster add-ons.",
			InventoryType:    "software",
			LabelPrefix:      "aws-eks-addon",
		}
	default:
		return ResourceMetadata{}
	}
}
