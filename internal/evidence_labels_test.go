package internal

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestEvidenceLabelKeysUseUnderscoreNotation(t *testing.T) {
	contexts := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "cluster",
			labels: BuildClusterEvidenceContext(types.Cluster{
				Name: aws.String("prod"),
				Arn:  aws.String("arn:aws:eks:eu-west-2:123456789012:cluster/prod"),
			}, "eu-west-2").Labels,
		},
		{
			name: "nodegroup",
			labels: BuildNodegroupEvidenceContext(types.Nodegroup{
				ClusterName:   aws.String("prod"),
				NodegroupName: aws.String("workers"),
				NodegroupArn:  aws.String("arn:aws:eks:eu-west-2:123456789012:nodegroup/prod/workers/id"),
			}, "eu-west-2").Labels,
		},
		{
			name: "addon",
			labels: BuildAddonEvidenceContext(types.Addon{
				ClusterName: aws.String("prod"),
				AddonName:   aws.String("vpc-cni"),
				AddonArn:    aws.String("arn:aws:eks:eu-west-2:123456789012:addon/prod/vpc-cni/id"),
			}, "eu-west-2").Labels,
		},
	}

	for _, context := range contexts {
		for key := range context.labels {
			if strings.Contains(key, "-") {
				t.Fatalf("%s evidence label key %q must use underscore notation", context.name, key)
			}
		}
	}
}
