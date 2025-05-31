package domain

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageRules struct {
	MaxWeight float64 `json:"max_weight"`
	Price     float64 `json:"price"`
}

type PackageConfig map[string][]PackageRules

func LoadPackageConfig(configPath string) (PackageConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s does not exist", configPath)
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config PackageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	for packageType, rules := range config {
		if len(rules) == 0 {
			return nil, fmt.Errorf("invalid config: package type %s has no rules", packageType)
		}
		for i, rule := range rules {
			if rule.Price < 0 {
				return nil, fmt.Errorf("invalid config: negative price %.2f for package type %s, rule %d", rule.Price, packageType, i)
			}
			if rule.MaxWeight < 0 {
				return nil, fmt.Errorf("invalid config: negative max_weight %.2f for package type %s, rule %d", rule.MaxWeight, packageType, i)
			}
		}
	}

	return config, nil
}

func (c PackageConfig) IsValidPackageType(packageType string) bool {
	if packageType == "" {
		return true
	}
	_, exists := c[packageType]
	return exists
}

func (c PackageConfig) GetRules(packageType string) ([]PackageRules, bool) {
	rules, exists := c[packageType]
	return rules, exists
}
