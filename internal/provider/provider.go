package provider

import (
	"context"
	"os"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type incidentGardenProvider struct{ version string }

type providerModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	Token        types.String `tfsdk:"token"`
	CSRFToken    types.String `tfsdk:"csrf_token"`
	Organization types.String `tfsdk:"organization"`
}

type providerData struct {
	Client              *client.Client
	DefaultOrganization string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &incidentGardenProvider{version: version} }
}

func (p *incidentGardenProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "incidentgarden"
	resp.Version = p.version
}

func (p *incidentGardenProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{Attributes: map[string]providerschema.Attribute{
		"endpoint":     providerschema.StringAttribute{Optional: true, Description: "Incident Garden API base URL. Defaults to INCIDENTGARDEN_ENDPOINT."},
		"token":        providerschema.StringAttribute{Optional: true, Sensitive: true, Description: "JWT access token. Defaults to INCIDENTGARDEN_TOKEN."},
		"csrf_token":   providerschema.StringAttribute{Optional: true, Sensitive: true, Description: "CSRF token sent as both cookie and header for mutations. Defaults to INCIDENTGARDEN_CSRF_TOKEN."},
		"organization": providerschema.StringAttribute{Optional: true, Description: "Default organization slug. Resources may override it."},
	}}
}

func envOrValue(v types.String, key, fallback string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (p *incidentGardenProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := envOrValue(config.Endpoint, "INCIDENTGARDEN_ENDPOINT", "http://localhost:8080")
	token := envOrValue(config.Token, "INCIDENTGARDEN_TOKEN", "")
	csrf := envOrValue(config.CSRFToken, "INCIDENTGARDEN_CSRF_TOKEN", "")
	if token == "" {
		resp.Diagnostics.AddError("Missing Incident Garden token", "Set token in the provider or INCIDENTGARDEN_TOKEN.")
		return
	}
	c, err := client.New(endpoint, token, csrf, nil)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	data := &providerData{Client: c, DefaultOrganization: envOrValue(config.Organization, "INCIDENTGARDEN_ORGANIZATION", "")}
	resp.DataSourceData, resp.ResourceData = data, data
}

func (p *incidentGardenProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewTeamResource, NewScheduleResource, NewEscalationPolicyResource, NewIntegrationResource, NewIntegrationPolicyResource}
}

func (p *incidentGardenProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

var _ provider.Provider = (*incidentGardenProvider)(nil)
