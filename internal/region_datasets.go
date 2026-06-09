package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/hashicorp/go-hclog"
)

type EKSClient interface {
	ListClusters(context.Context, *eks.ListClustersInput, ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	DescribeCluster(context.Context, *eks.DescribeClusterInput, ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
	ListNodegroups(context.Context, *eks.ListNodegroupsInput, ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
	DescribeNodegroup(context.Context, *eks.DescribeNodegroupInput, ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
	ListAddons(context.Context, *eks.ListAddonsInput, ...func(*eks.Options)) (*eks.ListAddonsOutput, error)
	DescribeAddon(context.Context, *eks.DescribeAddonInput, ...func(*eks.Options)) (*eks.DescribeAddonOutput, error)
}

type RegionDatasets struct {
	Clusters   []types.Cluster
	Nodegroups []types.Nodegroup
	Addons     []types.Addon
}

func CollectRegionDatasets(ctx context.Context, logger hclog.Logger, client EKSClient, requiredDatasets map[string]bool) (RegionDatasets, error) {
	if !requiresEKSClient(requiredDatasets) {
		return RegionDatasets{}, nil
	}
	if client == nil {
		return RegionDatasets{}, errors.New("eks client is required for requested region datasets")
	}

	clusterNames, err := listClusterNames(ctx, client)
	if err != nil {
		logger.Error("unable to list EKS clusters", "error", err)
		return RegionDatasets{}, err
	}

	clusters, err := describeClusters(ctx, client, clusterNames)
	if err != nil {
		logger.Error("unable to describe EKS clusters", "error", err)
		return RegionDatasets{}, err
	}

	datasets := RegionDatasets{
		Clusters: clusters,
	}

	if requiredDatasets["nodegroups"] {
		datasets.Nodegroups, err = collectNodegroups(ctx, client, clusterNames)
		if err != nil {
			logger.Error("unable to collect EKS node groups", "error", err)
			return RegionDatasets{}, err
		}
	}

	if requiredDatasets["addons"] {
		datasets.Addons, err = collectAddons(ctx, client, clusterNames)
		if err != nil {
			logger.Error("unable to collect EKS add-ons", "error", err)
			return RegionDatasets{}, err
		}
	}

	return datasets, nil
}

func listClusterNames(ctx context.Context, client EKSClient) ([]string, error) {
	names := make([]string, 0)
	var nextToken *string
	for {
		output, err := client.ListClusters(ctx, &eks.ListClustersInput{NextToken: nextToken})
		if err != nil {
			return nil, err
		}
		names = append(names, output.Clusters...)
		if output.NextToken == nil {
			return names, nil
		}
		nextToken = output.NextToken
	}
}

func describeClusters(ctx context.Context, client EKSClient, clusterNames []string) ([]types.Cluster, error) {
	clusters := make([]types.Cluster, 0, len(clusterNames))
	for _, clusterName := range clusterNames {
		output, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(clusterName)})
		if err != nil {
			return nil, fmt.Errorf("describe cluster %s: %w", clusterName, err)
		}
		if output.Cluster != nil {
			clusters = append(clusters, *output.Cluster)
		}
	}
	return clusters, nil
}

func collectNodegroups(ctx context.Context, client EKSClient, clusterNames []string) ([]types.Nodegroup, error) {
	nodegroups := make([]types.Nodegroup, 0)
	for _, clusterName := range clusterNames {
		nodegroupNames, err := listNodegroupNames(ctx, client, clusterName)
		if err != nil {
			return nil, err
		}
		for _, nodegroupName := range nodegroupNames {
			output, err := client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
				ClusterName:   aws.String(clusterName),
				NodegroupName: aws.String(nodegroupName),
			})
			if err != nil {
				return nil, fmt.Errorf("describe node group %s/%s: %w", clusterName, nodegroupName, err)
			}
			if output.Nodegroup != nil {
				nodegroups = append(nodegroups, *output.Nodegroup)
			}
		}
	}
	return nodegroups, nil
}

func listNodegroupNames(ctx context.Context, client EKSClient, clusterName string) ([]string, error) {
	names := make([]string, 0)
	var nextToken *string
	for {
		output, err := client.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list node groups for cluster %s: %w", clusterName, err)
		}
		names = append(names, output.Nodegroups...)
		if output.NextToken == nil {
			return names, nil
		}
		nextToken = output.NextToken
	}
}

func collectAddons(ctx context.Context, client EKSClient, clusterNames []string) ([]types.Addon, error) {
	addons := make([]types.Addon, 0)
	for _, clusterName := range clusterNames {
		addonNames, err := listAddonNames(ctx, client, clusterName)
		if err != nil {
			return nil, err
		}
		for _, addonName := range addonNames {
			output, err := client.DescribeAddon(ctx, &eks.DescribeAddonInput{
				ClusterName: aws.String(clusterName),
				AddonName:   aws.String(addonName),
			})
			if err != nil {
				return nil, fmt.Errorf("describe add-on %s/%s: %w", clusterName, addonName, err)
			}
			if output.Addon != nil {
				addons = append(addons, *output.Addon)
			}
		}
	}
	return addons, nil
}

func listAddonNames(ctx context.Context, client EKSClient, clusterName string) ([]string, error) {
	names := make([]string, 0)
	var nextToken *string
	for {
		output, err := client.ListAddons(ctx, &eks.ListAddonsInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list add-ons for cluster %s: %w", clusterName, err)
		}
		names = append(names, output.Addons...)
		if output.NextToken == nil {
			return names, nil
		}
		nextToken = output.NextToken
	}
}

func requiresEKSClient(requiredDatasets map[string]bool) bool {
	for _, datasetName := range []string{"clusters", "nodegroups", "addons"} {
		if requiredDatasets[datasetName] {
			return true
		}
	}
	return false
}
