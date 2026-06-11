package main

import (
	"reflect"
	"testing"
)

func TestSupportedPolicyBehaviorsOnlyIncludesEksResourceFamilies(t *testing.T) {
	expected := []string{"cluster", "nodegroup", "addon"}
	if actual := supportedPolicyBehaviors(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("supportedPolicyBehaviors() = %v, want exactly %v", actual, expected)
	}
}

func TestDefaultPolicyBehaviorMappingUsesNaturalEksBundleSplit(t *testing.T) {
	mapping := defaultPolicyBehaviorMapping()

	expected := map[string][]string{
		"aws-eks-policies":           {"cluster"},
		"aws-eks-nodegroup-policies": {"nodegroup"},
		"aws-eks-addon-policies":     {"addon"},
	}
	if !reflect.DeepEqual(mapping, expected) {
		t.Fatalf("defaultPolicyBehaviorMapping() = %v, want %v", mapping, expected)
	}
}

func TestBuildRequiredDatasetsReusesSharedClusterCollection(t *testing.T) {
	required := buildRequiredDatasets(map[string][]string{
		"cluster":   {"/tmp/cluster-policies"},
		"nodegroup": {"/tmp/nodegroup-policies"},
		"addon":     {"/tmp/addon-policies"},
	})

	for _, dataset := range []string{"clusters", "nodegroups", "addons"} {
		if !required[dataset] {
			t.Fatalf("expected %s to be required", dataset)
		}
	}
	if len(required) != 3 {
		t.Fatalf("required datasets = %v, want only clusters, nodegroups, addons", required)
	}
}

func TestBuildRequiredDatasetsForClusterPoliciesIncludesAddonsForClusterContext(t *testing.T) {
	required := buildRequiredDatasets(map[string][]string{
		"cluster": {"/tmp/cluster-policies"},
	})

	for _, dataset := range []string{"clusters", "addons"} {
		if !required[dataset] {
			t.Fatalf("expected %s to be required for cluster policies", dataset)
		}
	}
	if required["nodegroups"] {
		t.Fatal("nodegroups should not be required for cluster policies")
	}
}
