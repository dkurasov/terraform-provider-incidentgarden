package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	stringListType = types.ListType{ElemType: types.StringType}
	int64ListType  = types.ListType{ElemType: types.Int64Type}

	severityLevelsTypes = map[string]attr.Type{
		"critical": stringListType,
		"warning":  stringListType,
	}
	severityMappingTypes = map[string]attr.Type{
		"source_field": types.StringType,
		"mapping":      types.ObjectType{AttrTypes: severityLevelsTypes},
		"default":      types.StringType,
	}
	policyMatcherTypes = map[string]attr.Type{
		"key":   types.StringType,
		"op":    types.StringType,
		"value": types.StringType,
	}
	policyWindowTypes = map[string]attr.Type{
		"start":        types.StringType,
		"end":          types.StringType,
		"days_of_week": int64ListType,
	}
	policyConditionsTypes = map[string]attr.Type{
		"matchers":    types.ListType{ElemType: types.ObjectType{AttrTypes: policyMatcherTypes}},
		"severity_in": stringListType,
		"time_window": types.ObjectType{AttrTypes: policyWindowTypes},
	}
	policyActionValueTypes = map[string]attr.Type{
		"value":        types.StringType,
		"labels":       types.MapType{ElemType: types.StringType},
		"start":        types.StringType,
		"end":          types.StringType,
		"days_of_week": int64ListType,
	}
	policyActionTypes = map[string]attr.Type{
		"type":  types.StringType,
		"value": types.ObjectType{AttrTypes: policyActionValueTypes},
	}
)

func severityMappingAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Typed severity extraction and mapping configuration.",
		Attributes: map[string]schema.Attribute{
			"source_field": schema.StringAttribute{Optional: true, Computed: true},
			"mapping": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"critical": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
				"warning":  schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			}},
			"default": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func policyConditionsAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Typed conditions; every configured condition must match.",
		Attributes: map[string]schema.Attribute{
			"matchers": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{Required: true}, "op": schema.StringAttribute{Required: true}, "value": schema.StringAttribute{Optional: true},
			}}},
			"severity_in": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"time_window": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"start": schema.StringAttribute{Required: true}, "end": schema.StringAttribute{Required: true}, "days_of_week": schema.ListAttribute{Required: true, ElementType: types.Int64Type},
			}},
		},
	}
}

func policyActionAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "One typed policy action. type discriminates the shape of value.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{Required: true},
			"value": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"value":        schema.StringAttribute{Optional: true},
				"labels":       schema.MapAttribute{Optional: true, ElementType: types.StringType},
				"start":        schema.StringAttribute{Optional: true},
				"end":          schema.StringAttribute{Optional: true},
				"days_of_week": schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			}},
		},
	}
}

func objectValue(value types.Object) map[string]any {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	return attributesValue(value.Attributes())
}

func attributesValue(values map[string]attr.Value) map[string]any {
	out := make(map[string]any)
	for name, value := range values {
		if value.IsNull() || value.IsUnknown() {
			continue
		}
		switch typed := value.(type) {
		case types.String:
			out[name] = typed.ValueString()
		case types.Int64:
			out[name] = typed.ValueInt64()
		case types.Object:
			out[name] = attributesValue(typed.Attributes())
		case types.List:
			items := make([]any, 0, len(typed.Elements()))
			for _, item := range typed.Elements() {
				switch item := item.(type) {
				case types.String:
					items = append(items, item.ValueString())
				case types.Int64:
					items = append(items, item.ValueInt64())
				case types.Object:
					items = append(items, attributesValue(item.Attributes()))
				}
			}
			out[name] = items
		case types.Map:
			items := make(map[string]string, len(typed.Elements()))
			for key, item := range typed.Elements() {
				items[key] = item.(types.String).ValueString()
			}
			out[name] = items
		}
	}
	return out
}

func objectFromJSON(raw json.RawMessage, objectTypes map[string]attr.Type) (types.Object, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return types.ObjectNull(objectTypes), nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return types.ObjectNull(objectTypes), err
	}
	return objectFromMap(decoded, objectTypes), nil
}

func objectFromMap(decoded map[string]any, objectTypes map[string]attr.Type) types.Object {
	values := make(map[string]attr.Value, len(objectTypes))
	for name, valueType := range objectTypes {
		value, ok := decoded[name]
		if !ok || value == nil {
			values[name] = nullValue(valueType)
			continue
		}
		switch typed := valueType.(type) {
		case basetypes.StringType:
			values[name] = types.StringValue(value.(string))
		case basetypes.Int64Type:
			values[name] = types.Int64Value(int64(value.(float64)))
		case basetypes.ObjectType:
			values[name] = objectFromMap(value.(map[string]any), typed.AttrTypes)
		case basetypes.ListType:
			items := value.([]any)
			elements := make([]attr.Value, 0, len(items))
			for _, item := range items {
				switch elementType := typed.ElemType.(type) {
				case basetypes.StringType:
					elements = append(elements, types.StringValue(item.(string)))
				case basetypes.Int64Type:
					elements = append(elements, types.Int64Value(int64(item.(float64))))
				case basetypes.ObjectType:
					elements = append(elements, objectFromMap(item.(map[string]any), elementType.AttrTypes))
				}
			}
			values[name] = types.ListValueMust(typed.ElemType, elements)
		case basetypes.MapType:
			items := value.(map[string]any)
			elements := make(map[string]attr.Value, len(items))
			for key, item := range items {
				elements[key] = types.StringValue(item.(string))
			}
			values[name] = types.MapValueMust(typed.ElemType, elements)
		}
	}
	return types.ObjectValueMust(objectTypes, values)
}

func nullValue(valueType attr.Type) attr.Value {
	switch typed := valueType.(type) {
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.ObjectType:
		return types.ObjectNull(typed.AttrTypes)
	case basetypes.ListType:
		return types.ListNull(typed.ElemType)
	case basetypes.MapType:
		return types.MapNull(typed.ElemType)
	default:
		panic("unsupported structured value type")
	}
}
