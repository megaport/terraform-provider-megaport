package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMegaportProviderSchema_ManagedAccountUID(t *testing.T) {
	p := &megaportProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["managed_account_uid"]
	if !ok {
		t.Fatal("expected managed_account_uid attribute in provider schema")
	}

	strAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected managed_account_uid to be a StringAttribute, got %T", attr)
	}

	if !strAttr.Optional {
		t.Error("expected managed_account_uid to be optional")
	}
	if strAttr.Required {
		t.Error("expected managed_account_uid to not be required")
	}
	if !strings.Contains(strAttr.Description, "MEGAPORT_MANAGED_ACCOUNT_UID") {
		t.Errorf("expected managed_account_uid description to mention MEGAPORT_MANAGED_ACCOUNT_UID, got %q", strAttr.Description)
	}

	// An explicit empty string must be rejected so it can't silently blank
	// the MEGAPORT_MANAGED_ACCOUNT_UID env var; a non-empty value passes.
	for _, tc := range []struct {
		value       string
		wantError   bool
		description string
	}{
		{value: "", wantError: true, description: "empty string"},
		{value: "abc-123", wantError: false, description: "non-empty UID"},
	} {
		var diags int
		for _, v := range strAttr.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tc.value),
			}, resp)
			diags += resp.Diagnostics.ErrorsCount()
		}
		if tc.wantError && diags == 0 {
			t.Errorf("expected validation error for %s, got none", tc.description)
		}
		if !tc.wantError && diags != 0 {
			t.Errorf("expected no validation error for %s, got %d", tc.description, diags)
		}
	}
}
