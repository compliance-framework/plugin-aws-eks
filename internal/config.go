package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type PluginConfig struct {
	PolicyLabels map[string]string
	Regions      []string
}

func ParseConfig(raw map[string]string) (*PluginConfig, error) {
	config := &PluginConfig{}

	if v := strings.TrimSpace(raw["policy_labels"]); v != "" {
		if err := json.Unmarshal([]byte(v), &config.PolicyLabels); err != nil {
			return nil, fmt.Errorf("could not parse policy_labels: %w", err)
		}
	}

	config.Regions = parseRegions(raw)
	return config, nil
}

func parseRegions(raw map[string]string) []string {
	if regionsStr := strings.TrimSpace(raw["regions"]); regionsStr != "" {
		parts := strings.Split(regionsStr, ",")
		regions := make([]string, 0, len(parts))
		seen := make(map[string]bool)
		for _, part := range parts {
			region := strings.TrimSpace(part)
			if region == "" || seen[region] {
				continue
			}
			seen[region] = true
			regions = append(regions, region)
		}
		if len(regions) > 0 {
			return regions
		}
	}

	if region := strings.TrimSpace(raw["region"]); region != "" {
		return []string{region}
	}

	if region := strings.TrimSpace(os.Getenv("AWS_REGION")); region != "" {
		return []string{region}
	}

	return []string{"us-east-1"}
}
