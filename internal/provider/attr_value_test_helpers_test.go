package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// asTypesString asserts v is a types.String and returns its underlying value.
// Use in tests to safely extract strings from attr.Value maps/lists without
// tripping golangci-lint's forcetypeassert.
func asTypesString(t *testing.T, v attr.Value) string {
	t.Helper()
	s, ok := v.(types.String)
	require.True(t, ok, "value is not types.String, got %T", v)
	return s.ValueString()
}
