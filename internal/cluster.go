package internal

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/compliance-framework/agent/runner/proto"
)

func EvaluateClusterPolicies(deps EvaluationDependencies, policyPaths []string, clusters []types.Cluster, region string, datasets RegionDatasets) ResourceEvaluationErrors {
	return EvaluateResources(
		deps,
		policyPaths,
		clusters,
		func(cluster types.Cluster) ResourceEvidenceContext {
			return BuildClusterEvidenceContext(cluster, region)
		},
		func(cluster types.Cluster) (interface{}, error) {
			return BuildClusterPolicyInput(cluster, region, datasets)
		},
		func(cluster types.Cluster, err error) {
			deps.Logger.Error("unable to build EKS cluster policy input", "cluster_name", aws.ToString(cluster.Name), "region", region, "error", err)
		},
		func(evidences []*proto.Evidence, cluster types.Cluster) {
			PrefixEvidenceTitles(evidences, ClusterDisplayName(cluster))
		},
	)
}

func BuildClusterPolicyInput(cluster types.Cluster, region string, datasets RegionDatasets) (map[string]interface{}, error) {
	clusterValue, err := ToInterfaceMap(cluster)
	if err != nil {
		return nil, err
	}

	contextValue, err := ToInterfaceMap(buildClusterSupplementaryContext(cluster, region, datasets))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"cluster":         clusterValue,
		"cluster_context": contextValue,
	}, nil
}

func buildClusterSupplementaryContext(cluster types.Cluster, region string, datasets RegionDatasets) map[string]interface{} {
	clusterName := aws.ToString(cluster.Name)

	return map[string]interface{}{
		"current": map[string]interface{}{
			"cluster_name":                           clusterName,
			"cluster_arn":                            aws.ToString(cluster.Arn),
			"region":                                 region,
			"status":                                 string(cluster.Status),
			"version":                                aws.ToString(cluster.Version),
			"platform_version":                       aws.ToString(cluster.PlatformVersion),
			"health_issue_count":                     clusterHealthIssueCount(cluster),
			"tags_present":                           len(cluster.Tags) > 0,
			"endpoint_public_access":                 clusterEndpointPublicAccess(cluster),
			"endpoint_private_access":                clusterEndpointPrivateAccess(cluster),
			"public_access_cidrs":                    clusterPublicAccessCidrs(cluster),
			"enabled_control_plane_log_types":        enabledControlPlaneLogTypes(cluster),
			"authentication_mode":                    clusterAuthenticationMode(cluster),
			"bootstrap_creator_admin_permissions":    clusterBootstrapCreatorAdminPermissions(cluster),
			"secrets_encryption_configured":          clusterSecretsEncryptionConfigured(cluster),
			"secrets_encryption_customer_key_arns":   clusterSecretsEncryptionKeyARNs(cluster),
			"deletion_protection":                    aws.ToBool(cluster.DeletionProtection),
			"related_managed_nodegroup_count":        len(filterNodegroupsByCluster(datasets.Nodegroups, clusterName)),
			"related_managed_addon_count":            len(filterAddonsByCluster(datasets.Addons, clusterName)),
			"related_managed_addon_names":            addonNames(filterAddonsByCluster(datasets.Addons, clusterName)),
			"active_related_managed_addon_names":     activeAddonNames(filterAddonsByCluster(datasets.Addons, clusterName)),
			"related_managed_nodegroup_names":        nodegroupNames(filterNodegroupsByCluster(datasets.Nodegroups, clusterName)),
			"active_related_managed_nodegroup_names": activeNodegroupNames(filterNodegroupsByCluster(datasets.Nodegroups, clusterName)),
		},
		"addons":     filterAddonsByCluster(datasets.Addons, clusterName),
		"nodegroups": filterNodegroupsByCluster(datasets.Nodegroups, clusterName),
	}
}

func ClusterDisplayName(cluster types.Cluster) string {
	return aws.ToString(cluster.Name)
}

func clusterHealthIssueCount(cluster types.Cluster) int {
	if cluster.Health == nil {
		return 0
	}
	return len(cluster.Health.Issues)
}

func clusterPublicAccessCidrs(cluster types.Cluster) []string {
	if cluster.ResourcesVpcConfig == nil {
		return nil
	}
	return cluster.ResourcesVpcConfig.PublicAccessCidrs
}

func enabledControlPlaneLogTypes(cluster types.Cluster) []string {
	if cluster.Logging == nil {
		return nil
	}

	enabled := make([]string, 0)
	for _, setup := range cluster.Logging.ClusterLogging {
		if !aws.ToBool(setup.Enabled) {
			continue
		}
		for _, logType := range setup.Types {
			enabled = append(enabled, string(logType))
		}
	}
	return enabled
}

func clusterAuthenticationMode(cluster types.Cluster) string {
	if cluster.AccessConfig == nil {
		return ""
	}
	return string(cluster.AccessConfig.AuthenticationMode)
}

func clusterBootstrapCreatorAdminPermissions(cluster types.Cluster) bool {
	if cluster.AccessConfig == nil {
		return false
	}
	if cluster.AccessConfig.BootstrapClusterCreatorAdminPermissions == nil {
		return false
	}
	return *cluster.AccessConfig.BootstrapClusterCreatorAdminPermissions
}

func clusterSecretsEncryptionConfigured(cluster types.Cluster) bool {
	for _, encryptionConfig := range cluster.EncryptionConfig {
		for _, resource := range encryptionConfig.Resources {
			if resource == "secrets" {
				return true
			}
		}
	}
	return false
}

func clusterSecretsEncryptionKeyARNs(cluster types.Cluster) []string {
	keyARNs := make([]string, 0)
	for _, encryptionConfig := range cluster.EncryptionConfig {
		if encryptionConfig.Provider == nil {
			continue
		}
		for _, resource := range encryptionConfig.Resources {
			if resource == "secrets" && aws.ToString(encryptionConfig.Provider.KeyArn) != "" {
				keyARNs = append(keyARNs, aws.ToString(encryptionConfig.Provider.KeyArn))
				break
			}
		}
	}
	return keyARNs
}
