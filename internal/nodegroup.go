package internal

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/compliance-framework/agent/runner/proto"
)

func EvaluateNodegroupPolicies(deps EvaluationDependencies, policyPaths []string, nodegroups []types.Nodegroup, region string, datasets RegionDatasets) ResourceEvaluationErrors {
	return EvaluateResources(
		deps,
		policyPaths,
		nodegroups,
		func(nodegroup types.Nodegroup) ResourceEvidenceContext {
			return BuildNodegroupEvidenceContext(nodegroup, region)
		},
		func(nodegroup types.Nodegroup) (interface{}, error) {
			return BuildNodegroupPolicyInput(nodegroup, region, datasets)
		},
		func(nodegroup types.Nodegroup, err error) {
			deps.Logger.Error("unable to build EKS node group policy input", "cluster_name", aws.ToString(nodegroup.ClusterName), "nodegroup_name", aws.ToString(nodegroup.NodegroupName), "region", region, "error", err)
		},
		func(evidences []*proto.Evidence, nodegroup types.Nodegroup) {
			PrefixEvidenceTitles(evidences, NodegroupDisplayName(nodegroup))
		},
	)
}

func BuildNodegroupPolicyInput(nodegroup types.Nodegroup, region string, datasets RegionDatasets) (map[string]interface{}, error) {
	nodegroupValue, err := ToInterfaceMap(nodegroup)
	if err != nil {
		return nil, err
	}

	contextValue, err := ToInterfaceMap(buildNodegroupSupplementaryContext(nodegroup, region, datasets))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"nodegroup":         nodegroupValue,
		"nodegroup_context": contextValue,
	}, nil
}

func buildNodegroupSupplementaryContext(nodegroup types.Nodegroup, region string, datasets RegionDatasets) map[string]interface{} {
	clusterName := aws.ToString(nodegroup.ClusterName)
	scaling := nodegroupScalingSummary(nodegroup)

	return map[string]interface{}{
		"current": map[string]interface{}{
			"cluster_name":            clusterName,
			"nodegroup_name":          aws.ToString(nodegroup.NodegroupName),
			"nodegroup_arn":           aws.ToString(nodegroup.NodegroupArn),
			"region":                  region,
			"status":                  string(nodegroup.Status),
			"version":                 aws.ToString(nodegroup.Version),
			"release_version":         aws.ToString(nodegroup.ReleaseVersion),
			"capacity_type":           string(nodegroup.CapacityType),
			"ami_type":                string(nodegroup.AmiType),
			"health_issue_count":      nodegroupHealthIssueCount(nodegroup),
			"subnet_count":            len(nodegroup.Subnets),
			"tags_present":            len(nodegroup.Tags) > 0,
			"has_remote_access":       nodegroup.RemoteAccess != nil,
			"uses_launch_template":    nodegroup.LaunchTemplate != nil,
			"desired_size":            scaling["desired_size"],
			"min_size":                scaling["min_size"],
			"max_size":                scaling["max_size"],
			"scale_out_headroom":      scaling["scale_out_headroom"],
			"desired_at_or_above_min": scaling["desired_at_or_above_min"],
		},
		"cluster": findClusterByName(datasets.Clusters, clusterName),
	}
}

func NodegroupDisplayName(nodegroup types.Nodegroup) string {
	return aws.ToString(nodegroup.NodegroupName)
}

func nodegroupHealthIssueCount(nodegroup types.Nodegroup) int {
	if nodegroup.Health == nil {
		return 0
	}
	return len(nodegroup.Health.Issues)
}

func nodegroupScalingSummary(nodegroup types.Nodegroup) map[string]interface{} {
	summary := map[string]interface{}{
		"desired_size":            nil,
		"min_size":                nil,
		"max_size":                nil,
		"scale_out_headroom":      nil,
		"desired_at_or_above_min": nil,
	}
	if nodegroup.ScalingConfig == nil {
		return summary
	}

	desiredSize := aws.ToInt32(nodegroup.ScalingConfig.DesiredSize)
	minSize := aws.ToInt32(nodegroup.ScalingConfig.MinSize)
	maxSize := aws.ToInt32(nodegroup.ScalingConfig.MaxSize)
	summary["desired_size"] = desiredSize
	summary["min_size"] = minSize
	summary["max_size"] = maxSize
	summary["scale_out_headroom"] = maxSize - desiredSize
	summary["desired_at_or_above_min"] = desiredSize >= minSize
	return summary
}

func filterNodegroupsByCluster(nodegroups []types.Nodegroup, clusterName string) []types.Nodegroup {
	filtered := make([]types.Nodegroup, 0)
	for _, nodegroup := range nodegroups {
		if aws.ToString(nodegroup.ClusterName) == clusterName {
			filtered = append(filtered, nodegroup)
		}
	}
	return filtered
}

func nodegroupNames(nodegroups []types.Nodegroup) []string {
	names := make([]string, 0, len(nodegroups))
	for _, nodegroup := range nodegroups {
		if name := aws.ToString(nodegroup.NodegroupName); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func activeNodegroupNames(nodegroups []types.Nodegroup) []string {
	names := make([]string, 0, len(nodegroups))
	for _, nodegroup := range nodegroups {
		if nodegroup.Status == types.NodegroupStatusActive {
			if name := aws.ToString(nodegroup.NodegroupName); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
