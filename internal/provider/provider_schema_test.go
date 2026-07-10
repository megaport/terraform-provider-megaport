package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
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
}
