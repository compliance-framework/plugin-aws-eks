package internal

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestBuildNodegroupPolicyInputIncludesNodegroupContext(t *testing.T) {
	nodegroup := types.Nodegroup{
		ClusterName:    aws.String("prod"),
		NodegroupName:  aws.String("workers"),
		NodegroupArn:   aws.String("arn:aws:eks:eu-west-2:123456789012:nodegroup/prod/workers/id"),
		Status:         types.NodegroupStatusActive,
		Version:        aws.String("1.30"),
		ReleaseVersion: aws.String("1.30.1"),
		CapacityType:   types.CapacityTypesOnDemand,
		AmiType:        types.AMITypesAl2X8664,
		Subnets:        []string{"subnet-1", "subnet-2"},
		Tags:           map[string]string{"Environment": "prod"},
		ScalingConfig: &types.NodegroupScalingConfig{
			MinSize:     aws.Int32(2),
			DesiredSize: aws.Int32(3),
			MaxSize:     aws.Int32(5),
		},
	}

	input, err := BuildNodegroupPolicyInput(nodegroup, "eu-west-2", RegionDatasets{
		Clusters: []types.Cluster{{Name: aws.String("prod")}},
	})
	if err != nil {
		t.Fatalf("BuildNodegroupPolicyInput returned error: %v", err)
	}

	if _, ok := input["nodegroup"].(map[string]interface{}); !ok {
		t.Fatalf("input[nodegroup] should contain the raw node group map")
	}

	contextMap, ok := input["nodegroup_context"].(map[string]interface{})
	if !ok {
		t.Fatalf("input[nodegroup_context] has unexpected type %T", input["nodegroup_context"])
	}
	cur, ok := contextMap["current"].(map[string]interface{})
	if !ok {
		t.Fatalf("nodegroup_context[current] has unexpected type %T", contextMap["current"])
	}
	if cur["cluster_name"] != "prod" {
		t.Fatalf("current.cluster_name = %v, want prod", cur["cluster_name"])
	}
	if cur["subnet_count"] != float64(2) {
		t.Fatalf("current.subnet_count = %v, want 2", cur["subnet_count"])
	}
	if cur["scale_out_headroom"] != float64(2) {
		t.Fatalf("current.scale_out_headroom = %v, want 2", cur["scale_out_headroom"])
	}
	if cur["desired_at_or_above_min"] != true {
		t.Fatalf("current.desired_at_or_above_min = %v, want true", cur["desired_at_or_above_min"])
	}
	if contextMap["cluster"] == nil {
		t.Fatal("nodegroup_context.cluster should include the parent cluster when available")
	}
}
