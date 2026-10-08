package provider

import (
	"context"
	"fmt"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type membersDataSource struct{ provider *providerData }

type membersDataSourceModel struct {
	Organization types.String `tfsdk:"organization"`
	ByEmail      types.Map    `tfsdk:"by_email"`
}

func NewMembersDataSource() datasource.DataSource { return &membersDataSource{} }

func (d *membersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_members"
}

func (d *membersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all members of an organization and maps email addresses to account IDs. Requires org_admin or org_owner.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{Optional: true, Computed: true, Description: "Organization slug; defaults to the provider organization."},
			"by_email":     schema.MapAttribute{Computed: true, ElementType: types.StringType, Description: "Map from exact member email address to account ID."},
		},
	}
}

func (d *membersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *providerData, got %T", req.ProviderData))
		return
	}
	d.provider = data
}

func (d *membersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config membersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, err := organization(config.Organization, d.provider.DefaultOrganization)
	if err != nil {
		resp.Diagnostics.AddError("Missing organization", err.Error())
		return
	}
	members, err := client.Paginate(ctx, func(ctx context.Context, limit, offset int) ([]client.OrganizationMember, int, error) {
		return d.provider.Client.ListOrganizationMembersPage(ctx, org, limit, offset)
	}, 100)
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "list organization members", err)
		return
	}
	byEmail := make(map[string]string, len(members))
	for _, member := range members {
		if existing, duplicate := byEmail[member.Email]; duplicate && existing != member.ID {
			resp.Diagnostics.AddError("Duplicate organization member email", fmt.Sprintf("Email %q maps to multiple account IDs.", member.Email))
			return
		}
		byEmail[member.Email] = member.ID
	}
	config.Organization = types.StringValue(org)
	mapValue, diagnostics := types.MapValueFrom(ctx, types.StringType, byEmail)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.ByEmail = mapValue
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSource = (*membersDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*membersDataSource)(nil)
