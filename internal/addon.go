package internal

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/compliance-framework/agent/runner/proto"
)

func EvaluateAddonPolicies(deps EvaluationDependencies, policyPaths []string, addons []types.Addon, region string, datasets RegionDatasets) ResourceEvaluationErrors {
	return EvaluateResources(
		deps,
		policyPaths,
		addons,
		func(addon types.Addon) ResourceEvidenceContext {
			return BuildAddonEvidenceContext(addon, region)
		},
		func(addon types.Addon) (interface{}, error) {
			return BuildAddonPolicyInput(addon, region, datasets)
		},
		func(addon types.Addon, err error) {
			deps.Logger.Error("unable to build EKS add-on policy input", "cluster_name", aws.ToString(addon.ClusterName), "addon_name", aws.ToString(addon.AddonName), "region", region, "error", err)
		},
		func(evidences []*proto.Evidence, addon types.Addon) {
			PrefixEvidenceTitles(evidences, AddonDisplayName(addon))
		},
	)
}

func BuildAddonPolicyInput(addon types.Addon, region string, datasets RegionDatasets) (map[string]interface{}, error) {
	addonValue, err := ToInterfaceMap(addon)
	if err != nil {
		return nil, err
	}

	contextValue, err := ToInterfaceMap(buildAddonSupplementaryContext(addon, region, datasets))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"addon":         addonValue,
		"addon_context": contextValue,
	}, nil
}

func buildAddonSupplementaryContext(addon types.Addon, region string, datasets RegionDatasets) map[string]interface{} {
	clusterName := aws.ToString(addon.ClusterName)

	return map[string]interface{}{
		"current": map[string]interface{}{
			"cluster_name":             clusterName,
			"addon_name":               aws.ToString(addon.AddonName),
			"addon_arn":                aws.ToString(addon.AddonArn),
			"region":                   region,
			"status":                   string(addon.Status),
			"addon_version":            aws.ToString(addon.AddonVersion),
			"health_issue_count":       addonHealthIssueCount(addon),
			"owner":                    aws.ToString(addon.Owner),
			"publisher":                aws.ToString(addon.Publisher),
			"tags_present":             len(addon.Tags) > 0,
			"has_service_account_role": aws.ToString(addon.ServiceAccountRoleArn) != "",
		},
		"cluster": findClusterByName(datasets.Clusters, clusterName),
	}
}

func AddonDisplayName(addon types.Addon) string {
	return aws.ToString(addon.AddonName)
}

func addonHealthIssueCount(addon types.Addon) int {
	if addon.Health == nil {
		return 0
	}
	return len(addon.Health.Issues)
}

func filterAddonsByCluster(addons []types.Addon, clusterName string) []types.Addon {
	filtered := make([]types.Addon, 0)
	for _, addon := range addons {
		if aws.ToString(addon.ClusterName) == clusterName {
			filtered = append(filtered, addon)
		}
	}
	return filtered
}

func addonNames(addons []types.Addon) []string {
	names := make([]string, 0, len(addons))
	for _, addon := range addons {
		if name := aws.ToString(addon.AddonName); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func activeAddonNames(addons []types.Addon) []string {
	names := make([]string, 0, len(addons))
	for _, addon := range addons {
		if addon.Status == types.AddonStatusActive {
			if name := aws.ToString(addon.AddonName); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func findClusterByName(clusters []types.Cluster, clusterName string) *types.Cluster {
	for _, cluster := range clusters {
		if aws.ToString(cluster.Name) == clusterName {
			clusterCopy := cluster
			return &clusterCopy
		}
	}
	return nil
}
