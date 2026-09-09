package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func configureResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse, target **providerData) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *providerData, got %T", req.ProviderData))
		return
	}
	*target = data
}

func organization(configured types.String, fallback string) (string, error) {
	if !configured.IsNull() && !configured.IsUnknown() && configured.ValueString() != "" {
		return configured.ValueString(), nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("organization must be set on the resource or provider")
}

func importCompositeID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	org, id, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <organization-slug>/<resource-id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), org)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func parseImportID(value string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected <organization-slug>/<resource-id>")
	}
	return parts[0], parts[1], nil
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

func canonicalJSON(raw []byte, fallback string) (types.String, error) {
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte(fallback)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return types.StringNull(), err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return types.StringNull(), err
	}
	return types.StringValue(string(encoded)), nil
}

func rawJSON(value types.String, fallback string) (json.RawMessage, error) {
	if value.IsNull() || value.IsUnknown() || strings.TrimSpace(value.ValueString()) == "" {
		return json.RawMessage(fallback), nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value.ValueString()), &decoded); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(decoded)
	return json.RawMessage(encoded), err
}

func apiDiagnostic(diags *diag.Diagnostics, operation string, err error) {
	diags.AddError("Incident Garden API error", operation+": "+err.Error())
}
