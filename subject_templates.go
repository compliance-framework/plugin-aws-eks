package main

import "github.com/compliance-framework/agent/runner/proto"

func buildSubjectTemplates() []*proto.SubjectTemplate {
	return []*proto.SubjectTemplate{
		{
			Name:                "aws-eks-cluster",
			Type:                proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			TitleTemplate:       `AWS EKS cluster {{ .cluster_name }} in {{ .region }}`,
			DescriptionTemplate: `Amazon EKS cluster {{ .cluster_name }} in AWS region {{ .region }}.`,
			PurposeTemplate:     "Represents an AWS EKS cluster evaluated for control-plane compliance posture.",
			IdentityLabelKeys:   []string{"provider", "region", "cluster_name"},
			SelectorLabels:      selectorLabelsForType("cluster"),
			LabelSchema: labelSchema(
				label("provider", "Cloud provider for the evaluated resource"),
				label("type", "EKS plugin resource type"),
				label("region", "AWS region containing the EKS resource"),
				label("cluster_name", "AWS EKS cluster name"),
				label("cluster_arn", "AWS EKS cluster ARN"),
			),
		},
		{
			Name:                "aws-eks-nodegroup",
			Type:                proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			TitleTemplate:       `AWS EKS node group {{ .nodegroup_name }} for {{ .cluster_name }} in {{ .region }}`,
			DescriptionTemplate: `Amazon EKS managed node group {{ .nodegroup_name }} for cluster {{ .cluster_name }}.`,
			PurposeTemplate:     "Represents an AWS EKS managed node group evaluated for managed compute compliance posture.",
			IdentityLabelKeys:   []string{"provider", "region", "cluster_name", "nodegroup_name"},
			SelectorLabels:      selectorLabelsForType("nodegroup"),
			LabelSchema: labelSchema(
				label("provider", "Cloud provider for the evaluated resource"),
				label("type", "EKS plugin resource type"),
				label("region", "AWS region containing the EKS resource"),
				label("cluster_name", "AWS EKS cluster name"),
				label("nodegroup_name", "AWS EKS managed node group name"),
				label("nodegroup_arn", "AWS EKS managed node group ARN"),
			),
		},
		{
			Name:                "aws-eks-addon",
			Type:                proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			TitleTemplate:       `AWS EKS add-on {{ .addon_name }} for {{ .cluster_name }} in {{ .region }}`,
			DescriptionTemplate: `Amazon EKS managed add-on {{ .addon_name }} for cluster {{ .cluster_name }}.`,
			PurposeTemplate:     "Represents an AWS EKS managed add-on evaluated for add-on compliance posture.",
			IdentityLabelKeys:   []string{"provider", "region", "cluster_name", "addon_name"},
			SelectorLabels:      selectorLabelsForType("addon"),
			LabelSchema: labelSchema(
				label("provider", "Cloud provider for the evaluated resource"),
				label("type", "EKS plugin resource type"),
				label("region", "AWS region containing the EKS resource"),
				label("cluster_name", "AWS EKS cluster name"),
				label("addon_name", "AWS EKS managed add-on name"),
				label("addon_arn", "AWS EKS managed add-on ARN"),
			),
		},
	}
}

func selectorLabelsForType(resourceType string) []*proto.SubjectLabelSelector {
	return []*proto.SubjectLabelSelector{
		{
			Key:   "type",
			Value: resourceType,
		},
	}
}

func label(key string, description string) *proto.SubjectLabelSchema {
	return &proto.SubjectLabelSchema{
		Key:         key,
		Description: description,
	}
}

func labelSchema(labels ...*proto.SubjectLabelSchema) []*proto.SubjectLabelSchema {
	return labels
}
