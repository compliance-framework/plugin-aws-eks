package internal

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/hashicorp/go-hclog"
)

func TestCollectRegionDatasetsRequiresEKSClientForEKSDatasets(t *testing.T) {
	_, err := CollectRegionDatasets(context.Background(), hclog.NewNullLogger(), nil, map[string]bool{
		"clusters": true,
	})
	if err == nil || !strings.Contains(err.Error(), "eks client is required") {
		t.Fatalf("CollectRegionDatasets error = %v, want EKS client required error", err)
	}
}

func TestCollectRegionDatasetsAllowsNilClientWhenNoDatasetsRequired(t *testing.T) {
	if _, err := CollectRegionDatasets(context.Background(), hclog.NewNullLogger(), nil, nil); err != nil {
		t.Fatalf("CollectRegionDatasets returned error with no required datasets: %v", err)
	}
}

func TestCollectRegionDatasetsReusesClusterListForAddonCollection(t *testing.T) {
	client := &fakeEKSClient{
		listClustersOutputs: []*eks.ListClustersOutput{
			{Clusters: []string{"prod", "stage"}},
		},
		clusters: map[string]types.Cluster{
			"prod":  {Name: aws.String("prod")},
			"stage": {Name: aws.String("stage")},
		},
		addons: map[string][]types.Addon{
			"prod":  {{ClusterName: aws.String("prod"), AddonName: aws.String("vpc-cni")}},
			"stage": {{ClusterName: aws.String("stage"), AddonName: aws.String("coredns")}},
		},
	}

	datasets, err := CollectRegionDatasets(context.Background(), hclog.NewNullLogger(), client, map[string]bool{
		"clusters": true,
		"addons":   true,
	})
	if err != nil {
		t.Fatalf("CollectRegionDatasets returned error: %v", err)
	}

	if len(datasets.Clusters) != 2 || len(datasets.Addons) != 2 {
		t.Fatalf("datasets = %+v, want 2 clusters and 2 add-ons", datasets)
	}
	if client.listClustersCalls != 1 {
		t.Fatalf("ListClusters calls = %d, want 1", client.listClustersCalls)
	}
	if client.listNodegroupsCalls != 0 || client.describeNodegroupCalls != 0 {
		t.Fatalf("node group calls = list:%d describe:%d, want zero", client.listNodegroupsCalls, client.describeNodegroupCalls)
	}
	if client.listAddonsCalls != 2 || client.describeAddonCalls != 2 {
		t.Fatalf("add-on calls = list:%d describe:%d, want 2/2", client.listAddonsCalls, client.describeAddonCalls)
	}
}

func TestCollectRegionDatasetsHandlesClusterPagination(t *testing.T) {
	client := &fakeEKSClient{
		listClustersOutputs: []*eks.ListClustersOutput{
			{Clusters: []string{"prod"}, NextToken: aws.String("next")},
			{Clusters: []string{"stage"}},
		},
		clusters: map[string]types.Cluster{
			"prod":  {Name: aws.String("prod")},
			"stage": {Name: aws.String("stage")},
		},
	}

	datasets, err := CollectRegionDatasets(context.Background(), hclog.NewNullLogger(), client, map[string]bool{
		"clusters": true,
	})
	if err != nil {
		t.Fatalf("CollectRegionDatasets returned error: %v", err)
	}
	if len(datasets.Clusters) != 2 {
		t.Fatalf("cluster count = %d, want 2", len(datasets.Clusters))
	}
	if client.listClustersCalls != 2 {
		t.Fatalf("ListClusters calls = %d, want 2", client.listClustersCalls)
	}
}

type fakeEKSClient struct {
	listClustersOutputs []*eks.ListClustersOutput
	clusters            map[string]types.Cluster
	nodegroups          map[string][]types.Nodegroup
	addons              map[string][]types.Addon

	listClustersCalls      int
	describeClusterCalls   int
	listNodegroupsCalls    int
	describeNodegroupCalls int
	listAddonsCalls        int
	describeAddonCalls     int
}

func (f *fakeEKSClient) ListClusters(context.Context, *eks.ListClustersInput, ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
	f.listClustersCalls++
	index := f.listClustersCalls - 1
	if index >= len(f.listClustersOutputs) {
		return &eks.ListClustersOutput{}, nil
	}
	return f.listClustersOutputs[index], nil
}

func (f *fakeEKSClient) DescribeCluster(_ context.Context, input *eks.DescribeClusterInput, _ ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	f.describeClusterCalls++
	cluster := f.clusters[aws.ToString(input.Name)]
	return &eks.DescribeClusterOutput{Cluster: &cluster}, nil
}

func (f *fakeEKSClient) ListNodegroups(_ context.Context, input *eks.ListNodegroupsInput, _ ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
	f.listNodegroupsCalls++
	nodegroups := f.nodegroups[aws.ToString(input.ClusterName)]
	names := make([]string, 0, len(nodegroups))
	for _, nodegroup := range nodegroups {
		names = append(names, aws.ToString(nodegroup.NodegroupName))
	}
	return &eks.ListNodegroupsOutput{Nodegroups: names}, nil
}

func (f *fakeEKSClient) DescribeNodegroup(_ context.Context, input *eks.DescribeNodegroupInput, _ ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
	f.describeNodegroupCalls++
	for _, nodegroup := range f.nodegroups[aws.ToString(input.ClusterName)] {
		if aws.ToString(nodegroup.NodegroupName) == aws.ToString(input.NodegroupName) {
			nodegroupCopy := nodegroup
			return &eks.DescribeNodegroupOutput{Nodegroup: &nodegroupCopy}, nil
		}
	}
	return &eks.DescribeNodegroupOutput{}, nil
}

func (f *fakeEKSClient) ListAddons(_ context.Context, input *eks.ListAddonsInput, _ ...func(*eks.Options)) (*eks.ListAddonsOutput, error) {
	f.listAddonsCalls++
	addons := f.addons[aws.ToString(input.ClusterName)]
	names := make([]string, 0, len(addons))
	for _, addon := range addons {
		names = append(names, aws.ToString(addon.AddonName))
	}
	return &eks.ListAddonsOutput{Addons: names}, nil
}

func (f *fakeEKSClient) DescribeAddon(_ context.Context, input *eks.DescribeAddonInput, _ ...func(*eks.Options)) (*eks.DescribeAddonOutput, error) {
	f.describeAddonCalls++
	for _, addon := range f.addons[aws.ToString(input.ClusterName)] {
		if aws.ToString(addon.AddonName) == aws.ToString(input.AddonName) {
			addonCopy := addon
			return &eks.DescribeAddonOutput{Addon: &addonCopy}, nil
		}
	}
	return &eks.DescribeAddonOutput{}, nil
}
