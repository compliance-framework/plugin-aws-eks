package internal

import "testing"

func TestParseConfigResolvesRegionsFromCommaSeparatedConfig(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")

	config, err := ParseConfig(map[string]string{
		"regions": "eu-west-2, us-east-1, eu-west-2",
	})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	expected := []string{"eu-west-2", "us-east-1"}
	if len(config.Regions) != len(expected) {
		t.Fatalf("regions = %v, want %v", config.Regions, expected)
	}
	for i := range expected {
		if config.Regions[i] != expected[i] {
			t.Fatalf("regions = %v, want %v", config.Regions, expected)
		}
	}
}

func TestParseConfigFallsBackFromWhitespaceRegionToEnvironment(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-2")

	config, err := ParseConfig(map[string]string{"region": "   \t  "})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if len(config.Regions) != 1 || config.Regions[0] != "eu-west-2" {
		t.Fatalf("regions = %v, want [eu-west-2]", config.Regions)
	}
}

func TestParseConfigParsesPolicyLabels(t *testing.T) {
	config, err := ParseConfig(map[string]string{
		"policy_labels": `{"environment":"prod"}`,
	})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if config.PolicyLabels["environment"] != "prod" {
		t.Fatalf("policy label environment = %q, want prod", config.PolicyLabels["environment"])
	}
}
