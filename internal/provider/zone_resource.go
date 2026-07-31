package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexdns/terraform-provider-nexdns/internal/api"
)

var _ resource.Resource = &ZoneResource{}
var _ resource.ResourceWithImportState = &ZoneResource{}

type ZoneResource struct {
	client *api.Client
}

type ZoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	NSGroup     types.String `tfsdk:"ns_group"`
	MasterIP    types.String `tfsdk:"master_ip"`
	Status      types.String `tfsdk:"status"`
	Nameservers types.List   `tfsdk:"nameservers"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewZoneResource() resource.Resource {
	return &ZoneResource{}
}

func (r *ZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *ZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS zone on NexDNS.\n\n" +
			"`ns_group` is the one attribute that changes in place. Every other attribute forces a new zone, " +
			"which deletes the zone together with all of its records – including records Terraform does not manage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the zone, assigned by the platform. Pass it as `zone_id` to " +
					"`nexdns_record` and `nexdns_dnssec`.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Domain name of the zone, without a trailing dot, for example `example.com`. " +
					"The platform stores names lower-cased and, for an internationalised domain, in punycode; a " +
					"configuration that spells the same name differently is kept as written rather than reported as " +
					"drift. Changing this forces a new zone.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "`master` for a zone whose records are managed here, or `slave` for a " +
					"secondary zone that transfers from your own primary server at `master_ip`. Defaults to " +
					"`master`. Secondary zones require a plan that includes them – see the " +
					"[pricing page](https://nexdns.tech/pricing). Changing this forces a new zone.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Optional and Computed: the API assigns the account's default group
			// when the config omits one and returns it, so the attribute must be
			// able to take a value the plan did not set.
			//
			// Deliberately NOT RequiresReplace: the API moves a zone between
			// groups in place, and replacement would mean deleting the zone and
			// every record in it, including records Terraform does not manage.
			"ns_group": schema.StringAttribute{
				MarkdownDescription: "Slug of the nameserver group that serves the zone. Omit it to use the " +
					"default group on your account. The valid values are the groups available to your account, " +
					"listed in the control panel and returned by the API. Changing this moves the zone to the " +
					"other group in place: the zone keeps its identifier and all of its records, and " +
					"`nameservers` is refreshed. Update the delegation at your registrar to the new nameservers " +
					"afterwards.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"master_ip": schema.StringAttribute{
				MarkdownDescription: "Public IP address of the primary server a `slave` zone transfers from. " +
					"Required for `slave` zones, unused for `master`. Changing this forces a new zone.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the zone as the platform reports it, `active` for a zone that " +
					"is serving.",
				Computed: true,
			},
			"nameservers": schema.ListAttribute{
				MarkdownDescription: "Hostnames of the nameservers now serving this zone. Delegate the domain " +
					"to these at your registrar.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the zone was created, as reported by the API.",
				Computed:            true,
			},
		},
	}
}

func (r *ZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*api.Client)
}

func (r *ZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api.CreateZoneRequest{Name: data.Name.ValueString()}
	if !data.Type.IsNull() {
		createReq.Type = data.Type.ValueString()
	}
	if !data.MasterIP.IsNull() {
		createReq.MasterIP = data.MasterIP.ValueString()
	}
	if !data.NSGroup.IsNull() {
		// NS groups are referenced by slug; the API validates it.
		createReq.NSGroup = data.NSGroup.ValueString()
	}

	zone, err := r.client.CreateZone(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create zone", err.Error())
		return
	}

	data.ID = types.StringValue(zone.ID)
	data.Type = types.StringValue(zone.Type)
	data.Status = types.StringValue(zone.Status)

	// Read full zone details for nameservers
	r.readZoneIntoModel(zone.ID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A zone deleted outside Terraform drops out of state so the next plan can
	// recreate it, rather than making the configuration unplannable.
	if _, err := r.client.GetZone(data.ID.ValueString()); err != nil && api.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readZoneIntoModel(data.ID.ValueString(), &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update moves the zone to another nameserver group, the one mutable field on a
// zone. Every other attribute carries RequiresReplace, so a plan that reaches
// here changed ns_group and nothing else.
func (r *ZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	if !plan.NSGroup.IsUnknown() && plan.NSGroup.ValueString() != state.NSGroup.ValueString() {
		if _, err := r.client.MoveZone(plan.ID.ValueString(), plan.NSGroup.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("ns_group"),
				"Failed to move zone",
				err.Error(),
			)

			return
		}
	}

	r.readZoneIntoModel(plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZoneResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(context.Background(), &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteZone(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete zone", err.Error())
	}
}

func (r *ZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ZoneResourceModel
	data.ID = types.StringValue(req.ID)
	r.readZoneIntoModel(req.ID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) readZoneIntoModel(id string, data *ZoneResourceModel, diags *diag.Diagnostics) {
	zone, err := r.client.GetZone(id)
	if err != nil {
		diags.AddError("Failed to read zone", err.Error())
		return
	}

	data.ID = types.StringValue(zone.ID)
	// The API lowercases the name, drops the trailing dot and stores IDNs as
	// punycode. Writing that back over a configuration that spelled the name
	// differently ends the apply with "inconsistent result", so an equivalent
	// spelling is kept.
	data.Name = types.StringValue(preserveConfigured(data.Name.ValueString(), zone.Name))
	data.Type = types.StringValue(zone.Type)
	data.Status = types.StringValue(zone.Status)
	data.CreatedAt = types.StringValue(zone.CreatedAt)

	if zone.NSGroup != nil {
		data.NSGroup = types.StringValue(zone.NSGroup.Slug)
	}

	data.Nameservers, _ = types.ListValueFrom(context.Background(), types.StringType, zone.Nameservers)
}
