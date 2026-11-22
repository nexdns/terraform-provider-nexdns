package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexdns/terraform-provider-nexdns/internal/api"
)

var _ resource.Resource = &DNSSECResource{}
var _ resource.ResourceWithImportState = &DNSSECResource{}

type DNSSECResource struct {
	client *api.Client
}

type DNSSECResourceModel struct {
	ZoneID    types.String `tfsdk:"zone_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	DSRecords types.List   `tfsdk:"ds_records"`
}

func NewDNSSECResource() resource.Resource {
	return &DNSSECResource{}
}

func (r *DNSSECResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec"
}

func (r *DNSSECResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Signs a zone with DNSSEC. Creating the resource enables signing, destroying it " +
			"disables signing.\n\n" +
			"Keys are held by the platform, so the resource has nothing to update: it carries no editable " +
			"attribute, and changing `zone_id` replaces it. Requires a plan that includes DNSSEC (see the " +
			"[pricing page](https://nexdns.tech/pricing)).\n\n" +
			"The chain of trust is complete once the `ds_records` this resource exports are published at the " +
			"domain's registrar. Terraform does not do that step.",
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the zone to sign, as returned by `nexdns_zone.<name>.id` or " +
					"the `nexdns_zone` data source. Changing it forces a new resource.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the zone is signed. True for as long as the resource exists; if " +
					"signing is turned off elsewhere, the resource drops out of state and the next plan re-enables it.",
				Computed: true,
			},
			"ds_records": schema.ListAttribute{
				MarkdownDescription: "The DS records to publish at the domain's registrar to complete the chain " +
					"of trust.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DNSSECResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*api.Client)
}

func (r *DNSSECResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSSECResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := r.client.EnableDNSSEC(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to enable DNSSEC", err.Error())
		return
	}

	data.Enabled = types.BoolValue(status.Enabled)
	data.DSRecords, _ = types.ListValueFrom(ctx, types.StringType, status.DSRecords)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSSECResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSSECResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := r.client.GetDNSSEC(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read DNSSEC", err.Error())
		return
	}

	if !status.Enabled {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Enabled = types.BoolValue(true)
	data.DSRecords, _ = types.ListValueFrom(ctx, types.StringType, status.DSRecords)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSSECResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "DNSSEC resource does not support updates.")
}

func (r *DNSSECResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DNSSECResourceModel
	resp.Diagnostics.Append(req.State.Get(context.Background(), &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DisableDNSSEC(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to disable DNSSEC", err.Error())
	}
}

func (r *DNSSECResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data DNSSECResourceModel
	data.ZoneID = types.StringValue(req.ID)

	status, err := r.client.GetDNSSEC(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read DNSSEC", err.Error())
		return
	}

	data.Enabled = types.BoolValue(status.Enabled)
	data.DSRecords, _ = types.ListValueFrom(ctx, types.StringType, status.DSRecords)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
