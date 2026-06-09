package internal

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestBuildAddonPolicyInputIncludesAddonContext(t *testing.T) {
	addon := types.Addon{
		ClusterName:           aws.String("prod"),
		AddonName:             aws.String("vpc-cni"),
		AddonArn:              aws.String("arn:aws:eks:eu-west-2:123456789012:addon/prod/vpc-cni/id"),
		AddonVersion:          aws.String("v1.19.0-eksbuild.1"),
		Status:                types.AddonStatusActive,
		Owner:                 aws.String("aws"),
		Publisher:             aws.String("eks"),
		ServiceAccountRoleArn: aws.String("arn:aws:iam::123456789012:role/eks-addon"),
		Tags:                  map[string]string{"Environment": "prod"},
	}

	input, err := BuildAddonPolicyInput(addon, "eu-west-2", RegionDatasets{
		Clusters: []types.Cluster{{Name: aws.String("prod")}},
	})
	if err != nil {
		t.Fatalf("BuildAddonPolicyInput returned error: %v", err)
	}

	if _, ok := input["addon"].(map[string]interface{}); !ok {
		t.Fatalf("input[addon] should contain the raw add-on map")
	}

	contextMap := input["addon_context"].(map[string]interface{})
	current := contextMap["current"].(map[string]interface{})
	if current["cluster_name"] != "prod" {
		t.Fatalf("current.cluster_name = %v, want prod", current["cluster_name"])
	}
	if current["addon_name"] != "vpc-cni" {
		t.Fatalf("current.addon_name = %v, want vpc-cni", current["addon_name"])
	}
	if current["has_service_account_role"] != true {
		t.Fatalf("current.has_service_account_role = %v, want true", current["has_service_account_role"])
	}
	if contextMap["cluster"] == nil {
		t.Fatal("addon_context.cluster should include the parent cluster when available")
	}
}
