package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func identityAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "Server-generated resource identifier.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"organization": schema.StringAttribute{Optional: true, Computed: true, Description: "Organization slug; defaults to the provider organization.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"created_at":   schema.StringAttribute{Computed: true},
		"updated_at":   schema.StringAttribute{Computed: true},
	}
}

func mergeAttributes(base map[string]schema.Attribute, extra map[string]schema.Attribute) map[string]schema.Attribute {
	for name, attribute := range extra {
		base[name] = attribute
	}
	return base
}
