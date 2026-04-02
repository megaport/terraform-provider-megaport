package provider

import (
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// filterModel maps filter block schema data.
type filterModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

// matchesTags checks if resource tags match the specified tag filters.
func matchesTags(resourceTags map[string]string, tagFilters map[string]string) bool {
	for key, value := range tagFilters {
		if resourceTags[key] != value {
			return false
		}
	}
	return true
}

func matchesNamePattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []string, item int) bool {
	strItem := strconv.Itoa(item)
	for _, s := range slice {
		if s == strItem {
			return true
		}
	}
	return false
}

func containsBool(slice []string, item bool) bool {
	strItem := strconv.FormatBool(item)
	for _, s := range slice {
		if s == strItem {
			return true
		}
	}
	return false
}
