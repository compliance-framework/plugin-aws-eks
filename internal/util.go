package internal

import "encoding/json"

func StringAddressed(str string) *string {
	return &str
}

func MergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, values := range maps {
		for k, v := range values {
			result[k] = v
		}
	}
	return result
}

func ResolveRegions(config *PluginConfig) []string {
	if config == nil || len(config.Regions) == 0 {
		return []string{"us-east-1"}
	}
	return config.Regions
}

func ToInterfaceMap(value interface{}) (map[string]interface{}, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	return result, nil
}
