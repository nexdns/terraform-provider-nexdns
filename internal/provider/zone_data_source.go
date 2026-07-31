package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexdns/terraform-provider-nexdns/internal/api"
)

var _ datasource.DataSource = &ZoneDataSource{}

type ZoneDataSource struct {
	client *api.Client
}

type ZoneDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Status      types.String `tfsdk:"status"`
	Nameservers types.List   `tfsdk:"nameservers"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewZoneDataSource() datasource.DataSource {
	return &ZoneDataSource{}
}

func (d *ZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a zone that already exists on your account by its domain name, so that " +
			"records can be managed in a zone Terraform did not create.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the zone. Pass it as `zone_id` to `nexdns_record` and " +
					"`nexdns_dnssec`.",
				Computed: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Domain name to look up, as the platform stores it: lower-case, without a " +
					"trailing dot, and in punycode for an internationalised domain. Any other spelling does not " +
					"match and the lookup fails.",
				Required: true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the zone, `master` or `slave`.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the zone as the platform reports it, `active` for a zone that " +
					"is serving.",
				Computed: true,
			},
			"nameservers": schema.ListAttribute{
				MarkdownDescription: "Hostnames of the nameservers serving this zone.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the zone was created, as reported by the API.",
				Computed:            true,
			},
		},
	}
}

func (d *ZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*api.Client)
}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Find zone by name
	zoneItem, err := d.client.GetZoneByName(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Zone not found", err.Error())
		return
	}

	// Get full zone details
	zone, err := d.client.GetZone(zoneItem.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read zone", err.Error())
		return
	}

	data.ID = types.StringValue(zone.ID)
	data.Name = types.StringValue(zone.Name)
	data.Type = types.StringValue(zone.Type)
	data.Status = types.StringValue(zone.Status)
	data.CreatedAt = types.StringValue(zone.CreatedAt)
	data.Nameservers, _ = types.ListValueFrom(ctx, types.StringType, zone.Nameservers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
