package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// portModelFromV1RawState builds a V2 singlePortResourceModel from a V1 raw
// JSON state map. Fields that were removed in V2 are silently dropped.
func portModelFromV1RawState(ctx context.Context, raw map[string]json.RawMessage) (*singlePortResourceModel, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	model := &singlePortResourceModel{}

	model.UID = unmarshalStringAttr(raw, "product_uid")
	model.Name = unmarshalStringAttr(raw, "product_name")
	model.PortSpeed = unmarshalInt64Attr(raw, "port_speed")
	model.LocationID = unmarshalInt64Attr(raw, "location_id")
	model.MarketplaceVisibility = unmarshalBoolAttr(raw, "marketplace_visibility")
	model.CompanyUID = unmarshalStringAttr(raw, "company_uid")
	model.CostCentre = unmarshalStringAttr(raw, "cost_centre")
	model.ContractTermMonths = unmarshalInt64Attr(raw, "contract_term_months")
	model.DiversityZone = unmarshalStringAttr(raw, "diversity_zone")
	model.PromoCode = unmarshalStringAttr(raw, "promo_code")

	resourcesObj, resDiags := unmarshalPortResources(ctx, raw)
	diags.Append(resDiags...)
	model.Resources = resourcesObj

	tagsMap, tagDiags := unmarshalResourceTags(ctx, raw)
	diags.Append(tagDiags...)
	model.ResourceTags = tagsMap

	return model, diags
}

// lagPortModelFromV1RawState builds a V2 lagPortResourceModel from a V1 raw
// JSON state map. Fields that were removed in V2 are silently dropped.
func lagPortModelFromV1RawState(ctx context.Context, raw map[string]json.RawMessage) (*lagPortResourceModel, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	model := &lagPortResourceModel{}

	model.UID = unmarshalStringAttr(raw, "product_uid")
	model.Name = unmarshalStringAttr(raw, "product_name")
	model.PortSpeed = unmarshalInt64Attr(raw, "port_speed")
	model.LocationID = unmarshalInt64Attr(raw, "location_id")
	model.MarketplaceVisibility = unmarshalBoolAttr(raw, "marketplace_visibility")
	model.CompanyUID = unmarshalStringAttr(raw, "company_uid")
	model.CostCentre = unmarshalStringAttr(raw, "cost_centre")
	model.ContractTermMonths = unmarshalInt64Attr(raw, "contract_term_months")
	model.DiversityZone = unmarshalStringAttr(raw, "diversity_zone")
	model.PromoCode = unmarshalStringAttr(raw, "promo_code")
	model.LagCount = unmarshalInt64Attr(raw, "lag_count")

	lagUIDs, lagDiags := unmarshalLagPortUIDs(raw)
	diags.Append(lagDiags...)
	model.LagPortUIDs = lagUIDs

	resourcesObj, resDiags := unmarshalPortResources(ctx, raw)
	diags.Append(resDiags...)
	model.Resources = resourcesObj

	tagsMap, tagDiags := unmarshalResourceTags(ctx, raw)
	diags.Append(tagDiags...)
	model.ResourceTags = tagsMap

	return model, diags
}

// unmarshalStringAttr extracts a string attribute from raw JSON state.
// Returns a null string if the key is missing or the value is JSON null.
func unmarshalStringAttr(raw map[string]json.RawMessage, key string) types.String {
	v, ok := raw[key]
	if !ok || isJSONNull(v) {
		return types.StringNull()
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// unmarshalInt64Attr extracts an int64 attribute from raw JSON state.
// Handles both JSON numbers and null values.
func unmarshalInt64Attr(raw map[string]json.RawMessage, key string) types.Int64 {
	v, ok := raw[key]
	if !ok || isJSONNull(v) {
		return types.Int64Null()
	}
	var n json.Number
	if err := json.Unmarshal(v, &n); err != nil {
		return types.Int64Null()
	}
	i, err := n.Int64()
	if err != nil {
		return types.Int64Null()
	}
	return types.Int64Value(i)
}

// unmarshalBoolAttr extracts a bool attribute from raw JSON state.
func unmarshalBoolAttr(raw map[string]json.RawMessage, key string) types.Bool {
	v, ok := raw[key]
	if !ok || isJSONNull(v) {
		return types.BoolNull()
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}

// unmarshalPortResources extracts the resources nested object from raw JSON state.
func unmarshalPortResources(ctx context.Context, raw map[string]json.RawMessage) (types.Object, diag.Diagnostics) {
	v, ok := raw["resources"]
	if !ok || isJSONNull(v) {
		return types.ObjectNull(portResourcesAttrs), nil
	}

	var resources struct {
		Interface *struct {
			Demarcation *string `json:"demarcation"`
			Up          *int64  `json:"up"`
		} `json:"interface"`
	}
	if err := json.Unmarshal(v, &resources); err != nil {
		return types.ObjectNull(portResourcesAttrs), nil
	}

	if resources.Interface == nil {
		return types.ObjectNull(portResourcesAttrs), nil
	}

	ifaceModel := &portInterfaceModel{
		Demarcation: types.StringValue(""),
		Up:          types.Int64Value(0),
	}
	if resources.Interface.Demarcation != nil {
		ifaceModel.Demarcation = types.StringValue(*resources.Interface.Demarcation)
	}
	if resources.Interface.Up != nil {
		ifaceModel.Up = types.Int64Value(*resources.Interface.Up)
	}

	ifaceObj, diags := types.ObjectValueFrom(ctx, portInterfaceAttrs, ifaceModel)
	if diags.HasError() {
		return types.ObjectNull(portResourcesAttrs), diags
	}

	resModel := &portResourcesModel{Interface: ifaceObj}
	return types.ObjectValueFrom(ctx, portResourcesAttrs, resModel)
}

// unmarshalResourceTags extracts the resource_tags map from raw JSON state.
func unmarshalResourceTags(_ context.Context, raw map[string]json.RawMessage) (types.Map, diag.Diagnostics) {
	v, ok := raw["resource_tags"]
	if !ok || isJSONNull(v) {
		return types.MapNull(types.StringType), nil
	}

	var tags map[string]string
	if err := json.Unmarshal(v, &tags); err != nil {
		return types.MapNull(types.StringType), nil
	}

	if len(tags) == 0 {
		return types.MapNull(types.StringType), nil
	}

	elements := make(map[string]attr.Value, len(tags))
	for k, val := range tags {
		elements[k] = types.StringValue(val)
	}
	return types.MapValue(types.StringType, elements)
}

// unmarshalLagPortUIDs extracts the lag_port_uids list from raw JSON state.
func unmarshalLagPortUIDs(raw map[string]json.RawMessage) (types.List, diag.Diagnostics) {
	v, ok := raw["lag_port_uids"]
	if !ok || isJSONNull(v) {
		return types.ListNull(types.StringType), nil
	}

	var uids []string
	if err := json.Unmarshal(v, &uids); err != nil {
		return types.ListNull(types.StringType), nil
	}

	if len(uids) == 0 {
		return types.ListNull(types.StringType), nil
	}

	vals := make([]attr.Value, len(uids))
	for i, uid := range uids {
		vals[i] = types.StringValue(uid)
	}
	return types.ListValue(types.StringType, vals)
}

// isJSONNull reports whether the raw JSON value is null.
func isJSONNull(v json.RawMessage) bool {
	return len(v) == 4 && string(v) == "null"
}
