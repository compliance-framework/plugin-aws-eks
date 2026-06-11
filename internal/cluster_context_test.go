package internal

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestBuildClusterPolicyInputIncludesClusterContext(t *testing.T) {
	bootstrapAdmin := false
	deletionProtection := true
	cluster := types.Cluster{
		Name:               aws.String("prod"),
		Arn:                aws.String("arn:aws:eks:eu-west-2:123456789012:cluster/prod"),
		Status:             types.ClusterStatusActive,
		Version:            aws.String("1.30"),
		PlatformVersion:    aws.String("eks.12"),
		Tags:               map[string]string{"Environment": "prod"},
		DeletionProtection: &deletionProtection,
		AccessConfig: &types.AccessConfigResponse{
			AuthenticationMode:                      types.AuthenticationModeApiAndConfigMap,
			BootstrapClusterCreatorAdminPermissions: &bootstrapAdmin,
		},
		ResourcesVpcConfig: &types.VpcConfigResponse{
			EndpointPublicAccess:  true,
			EndpointPrivateAccess: true,
			PublicAccessCidrs:     []string{"10.0.0.0/8"},
		},
		Logging: &types.Logging{
			ClusterLogging: []types.LogSetup{
				{Enabled: aws.Bool(true), Types: []types.LogType{types.LogTypeAudit, types.LogTypeAuthenticator}},
				{Enabled: aws.Bool(false), Types: []types.LogType{types.LogTypeApi}},
			},
		},
		EncryptionConfig: []types.EncryptionConfig{
			{
				Resources: []string{"secrets"},
				Provider:  &types.Provider{KeyArn: aws.String("arn:aws:kms:eu-west-2:123456789012:key/abc")},
			},
		},
	}

	input, err := BuildClusterPolicyInput(cluster, "eu-west-2", RegionDatasets{
		Addons: []types.Addon{
			{ClusterName: aws.String("prod"), AddonName: aws.String("vpc-cni"), Status: types.AddonStatusActive},
			{ClusterName: aws.String("prod"), AddonName: aws.String("coredns"), Status: types.AddonStatusDegraded},
			{ClusterName: aws.String("other"), AddonName: aws.String("kube-proxy"), Status: types.AddonStatusActive},
		},
		Nodegroups: []types.Nodegroup{
			{ClusterName: aws.String("prod"), NodegroupName: aws.String("workers"), Status: types.NodegroupStatusActive},
		},
	})
	if err != nil {
		t.Fatalf("BuildClusterPolicyInput returned error: %v", err)
	}

	if _, ok := input["cluster"].(map[string]interface{}); !ok {
		t.Fatalf("input[cluster] should contain the raw cluster map")
	}

	contextMap, ok := input["cluster_context"].(map[string]interface{})
	if !ok {
		t.Fatalf("input[cluster_context] has unexpected type %T", input["cluster_context"])
	}
	current, ok := contextMap["current"].(map[string]interface{})
	if !ok {
		t.Fatalf("cluster_context[current] has unexpected type %T", contextMap["current"])
	}
	if current["cluster_name"] != "prod" {
		t.Fatalf("current.cluster_name = %v, want prod", current["cluster_name"])
	}
	if current["authentication_mode"] != "API_AND_CONFIG_MAP" {
		t.Fatalf("current.authentication_mode = %v, want API_AND_CONFIG_MAP", current["authentication_mode"])
	}
	if current["secrets_encryption_configured"] != true {
		t.Fatalf("current.secrets_encryption_configured = %v, want true", current["secrets_encryption_configured"])
	}
	if current["related_managed_addon_count"] != float64(2) {
		t.Fatalf("current.related_managed_addon_count = %v, want 2", current["related_managed_addon_count"])
	}

	activeAddons, ok := current["active_related_managed_addon_names"].([]interface{})
	if !ok {
		t.Fatalf("current[active_related_managed_addon_names] has unexpected type %T", current["active_related_managed_addon_names"])
	}
	if len(activeAddons) != 1 || activeAddons[0] != "vpc-cni" {
		t.Fatalf("active add-ons = %v, want [vpc-cni]", activeAddons)
	}
}
