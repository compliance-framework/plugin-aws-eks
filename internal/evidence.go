package internal

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/compliance-framework/agent/runner/proto"
)

func BuildClusterEvidenceContext(cluster types.Cluster, region string) ResourceEvidenceContext {
	metadata := GetResourceMetadata(ResourceTypeCluster)
	clusterName := aws.ToString(cluster.Name)
	clusterArn := aws.ToString(cluster.Arn)
	inventoryID := fmt.Sprintf("%s/%s/%s", metadata.LabelPrefix, region, clusterName)

	labels := map[string]string{
		"provider":     "aws",
		"type":         string(ResourceTypeCluster),
		"region":       region,
		"cluster_name": clusterName,
		"cluster_arn":  clusterArn,
		"resource_arn": clusterArn,
	}

	return NewResourceEvidenceContext(
		labels,
		resourceSubjects(metadata, inventoryID),
		resourceComponents(metadata),
		[]*proto.InventoryItem{
			{
				Identifier:  inventoryID,
				Type:        metadata.InventoryType,
				Title:       fmt.Sprintf("%s [%s]", metadata.ComponentTitle, clusterName),
				Description: fmt.Sprintf("Amazon EKS cluster %s in region %s.", clusterName, region),
				Props: []*proto.Property{
					{Name: "cluster-name", Value: clusterName},
					{Name: "cluster-arn", Value: clusterArn},
					{Name: "region", Value: region},
					{Name: "status", Value: string(cluster.Status)},
					{Name: "version", Value: aws.ToString(cluster.Version)},
					{Name: "endpoint-public-access", Value: strconv.FormatBool(clusterEndpointPublicAccess(cluster))},
					{Name: "endpoint-private-access", Value: strconv.FormatBool(clusterEndpointPrivateAccess(cluster))},
				},
				ImplementedComponents: implementedComponents(metadata),
			},
		},
	)
}

func BuildNodegroupEvidenceContext(nodegroup types.Nodegroup, region string) ResourceEvidenceContext {
	metadata := GetResourceMetadata(ResourceTypeNodegroup)
	clusterName := aws.ToString(nodegroup.ClusterName)
	nodegroupName := aws.ToString(nodegroup.NodegroupName)
	nodegroupArn := aws.ToString(nodegroup.NodegroupArn)
	inventoryID := fmt.Sprintf("%s/%s/%s/%s", metadata.LabelPrefix, region, clusterName, nodegroupName)

	labels := map[string]string{
		"provider":       "aws",
		"type":           string(ResourceTypeNodegroup),
		"region":         region,
		"cluster_name":   clusterName,
		"nodegroup_name": nodegroupName,
		"nodegroup_arn":  nodegroupArn,
		"resource_arn":   nodegroupArn,
	}

	return NewResourceEvidenceContext(
		labels,
		resourceSubjects(metadata, inventoryID),
		resourceComponents(metadata),
		[]*proto.InventoryItem{
			{
				Identifier:  inventoryID,
				Type:        metadata.InventoryType,
				Title:       fmt.Sprintf("%s [%s/%s]", metadata.ComponentTitle, clusterName, nodegroupName),
				Description: fmt.Sprintf("Amazon EKS managed node group %s for cluster %s in region %s.", nodegroupName, clusterName, region),
				Props: []*proto.Property{
					{Name: "cluster-name", Value: clusterName},
					{Name: "nodegroup-name", Value: nodegroupName},
					{Name: "nodegroup-arn", Value: nodegroupArn},
					{Name: "region", Value: region},
					{Name: "status", Value: string(nodegroup.Status)},
					{Name: "version", Value: aws.ToString(nodegroup.Version)},
					{Name: "subnet-count", Value: strconv.Itoa(len(nodegroup.Subnets))},
				},
				ImplementedComponents: implementedComponents(metadata),
			},
		},
	)
}

func BuildAddonEvidenceContext(addon types.Addon, region string) ResourceEvidenceContext {
	metadata := GetResourceMetadata(ResourceTypeAddon)
	clusterName := aws.ToString(addon.ClusterName)
	addonName := aws.ToString(addon.AddonName)
	addonArn := aws.ToString(addon.AddonArn)
	inventoryID := fmt.Sprintf("%s/%s/%s/%s", metadata.LabelPrefix, region, clusterName, addonName)

	labels := map[string]string{
		"provider":     "aws",
		"type":         string(ResourceTypeAddon),
		"region":       region,
		"cluster_name": clusterName,
		"addon_name":   addonName,
		"addon_arn":    addonArn,
		"resource_arn": addonArn,
	}

	return NewResourceEvidenceContext(
		labels,
		resourceSubjects(metadata, inventoryID),
		resourceComponents(metadata),
		[]*proto.InventoryItem{
			{
				Identifier:  inventoryID,
				Type:        metadata.InventoryType,
				Title:       fmt.Sprintf("%s [%s/%s]", metadata.ComponentTitle, clusterName, addonName),
				Description: fmt.Sprintf("Amazon EKS managed add-on %s for cluster %s in region %s.", addonName, clusterName, region),
				Props: []*proto.Property{
					{Name: "cluster-name", Value: clusterName},
					{Name: "addon-name", Value: addonName},
					{Name: "addon-arn", Value: addonArn},
					{Name: "region", Value: region},
					{Name: "status", Value: string(addon.Status)},
					{Name: "version", Value: aws.ToString(addon.AddonVersion)},
				},
				ImplementedComponents: implementedComponents(metadata),
			},
		},
	)
}

func resourceComponents(metadata ResourceMetadata) []*proto.Component {
	return []*proto.Component{
		{
			Identifier:  metadata.ComponentID,
			Type:        metadata.ComponentType,
			Title:       metadata.ComponentTitle,
			Description: metadata.ComponentDesc,
			Purpose:     metadata.ComponentPurpose,
		},
	}
}

func resourceSubjects(metadata ResourceMetadata, inventoryID string) []*proto.Subject {
	return []*proto.Subject{
		{
			Type:       proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			Identifier: metadata.ComponentID,
		},
		{
			Type:       proto.SubjectType_SUBJECT_TYPE_INVENTORY_ITEM,
			Identifier: inventoryID,
		},
	}
}

func implementedComponents(metadata ResourceMetadata) []*proto.InventoryItemImplementedComponent {
	return []*proto.InventoryItemImplementedComponent{
		{
			Identifier: metadata.ComponentID,
		},
	}
}

func clusterEndpointPublicAccess(cluster types.Cluster) bool {
	if cluster.ResourcesVpcConfig == nil {
		return false
	}
	return cluster.ResourcesVpcConfig.EndpointPublicAccess
}

func clusterEndpointPrivateAccess(cluster types.Cluster) bool {
	if cluster.ResourcesVpcConfig == nil {
		return false
	}
	return cluster.ResourcesVpcConfig.EndpointPrivateAccess
}
