package provider

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type jsonStringType struct{ basetypes.StringType }

func (jsonStringType) Equal(other attr.Type) bool {
	_, ok := other.(jsonStringType)
	return ok
}

func (jsonStringType) String() string { return "provider.JSONStringType" }

func (jsonStringType) ValueFromString(_ context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return jsonStringValue{StringValue: value}, nil
}

func (jsonStringType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := (basetypes.StringType{}).ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	return jsonStringValue{StringValue: converted.(basetypes.StringValue)}, nil
}

func (jsonStringType) ValueType(context.Context) attr.Value {
	return jsonStringValue{}
}

type jsonStringValue struct{ basetypes.StringValue }

func (jsonStringValue) Type(context.Context) attr.Type { return jsonStringType{} }

func (value jsonStringValue) StringSemanticEquals(_ context.Context, other basetypes.StringValuable) (bool, diag.Diagnostics) {
	otherValue, ok := other.(jsonStringValue)
	if !ok {
		return false, nil
	}
	var left, right any
	if err := json.Unmarshal([]byte(value.ValueString()), &left); err != nil {
		return false, nil
	}
	if err := json.Unmarshal([]byte(otherValue.ValueString()), &right); err != nil {
		return false, nil
	}
	return reflect.DeepEqual(left, right), nil
}

type validJSONValidator struct{}

func (validJSONValidator) Description(context.Context) string { return "value must contain valid JSON" }

func (validJSONValidator) MarkdownDescription(ctx context.Context) string {
	return validJSONValidator{}.Description(ctx)
}

func (validJSONValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var value any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", err.Error())
	}
}

var _ basetypes.StringTypable = jsonStringType{}
var _ basetypes.StringValuableWithSemanticEquals = jsonStringValue{}
var _ validator.String = validJSONValidator{}
