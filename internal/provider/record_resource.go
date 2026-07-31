package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexdns/terraform-provider-nexdns/internal/api"
)

var _ resource.Resource = &RecordResource{}
var _ resource.ResourceWithImportState = &RecordResource{}

type RecordResource struct {
	client *api.Client
}

type RecordResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ZoneID       types.String `tfsdk:"zone_id"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Content      types.String `tfsdk:"content"`
	TTL          types.Int64  `tfsdk:"ttl"`
	Priority     types.Int64  `tfsdk:"priority"`
	Weight       types.Int64  `tfsdk:"weight"`
	Port         types.Int64  `tfsdk:"port"`
	Flags        types.Int64  `tfsdk:"flags"`
	Tag          types.String `tfsdk:"tag"`
	KeyTag       types.Int64  `tfsdk:"keytag"`
	Algorithm    types.Int64  `tfsdk:"algorithm"`
	DigestType   types.Int64  `tfsdk:"digest_type"`
	Usage        types.Int64  `tfsdk:"usage"`
	Selector     types.Int64  `tfsdk:"selector"`
	MatchingType types.Int64  `tfsdk:"matching_type"`
}

func NewRecordResource() resource.Resource {
	return &RecordResource{}
}

func (r *RecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *RecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single value in a DNS record set on NexDNS.\n\n" +
			"A record set is every value that shares a name and a type – the two `A` records behind one hostname " +
			"are one set of two values, and each value is one `nexdns_record`. The TTL is a property of the set " +
			"rather than of the value, so give every record in a set the same `ttl`; see the `ttl` attribute.\n\n" +
			"The zone's own `NS` records at the apex come from its nameserver group and are not managed here.",
		Attributes: map[string]schema.Attribute{
			// No UseStateForUnknown here: a record id is derived from its content,
			// so editing the content gives the record a new id. Pinning the id to
			// the prior state made every content change fail the apply with
			// "Provider produced inconsistent result after apply" even though the
			// change had already landed. Leaving it unknown lets the id from the
			// update response become the new state.
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the record. It is derived from the record's content, so " +
					"editing `content` gives the record a new id – expected, and not a replacement.",
				Computed: true,
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the zone that holds the record, as returned by " +
					"`nexdns_zone.<name>.id` or the `nexdns_zone` data source. Changing it forces a new record.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Record type: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA`, `PTR`, " +
					"`ALIAS`, `DNAME`, `DS` or `TLSA`. Changing it forces a new record.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Record name relative to the zone – `www`, `_dmarc`, or `@` for the zone " +
					"apex. Defaults to the apex. Labels are stored lower-cased, and in punycode for an " +
					"internationalised name; a configuration that spells the same name differently is kept as " +
					"written rather than reported as drift.",
				Optional: true,
				Computed: true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The record's primary value, and only that value: the address for `A` and " +
					"`AAAA`, the target hostname for `CNAME`, `NS`, `PTR`, `ALIAS`, `DNAME`, `MX` and `SRV`, the " +
					"text for `TXT` (unquoted – the quoting is applied for you), the issuer for `CAA`, the digest " +
					"for `DS`, the certificate association data for `TLSA`. Everything else a record type carries " +
					"goes in its own attribute, never inside `content`: an `MX` is " +
					"`content = \"mail.example.com\"` with `priority = 10`, not " +
					"`content = \"10 mail.example.com\"`.",
				Required: true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live, in seconds. A TTL belongs to the record set, not to the " +
					"single value: setting it here retimes every record that shares this record's `name` and " +
					"`type`. Give every record in a set the same `ttl`, or they will take turns changing it on " +
					"each apply. Left unset, an existing set keeps its TTL, a new set gets the platform default, " +
					"and the effective value is read back into state.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"priority": optionalComputedInt64("Preference of an `MX` record, or priority of an `SRV` record. " +
				"Lower is preferred. An `MX` record left without one is stored with priority 10."),
			"weight": optionalComputedInt64("Relative weight of an `SRV` record among targets of equal priority."),
			"port":   optionalComputedInt64("TCP or UDP port the service of an `SRV` record listens on."),
			"flags":  optionalComputedInt64("Flags octet of a `CAA` record; `0` for an ordinary, non-critical property."),
			"tag": optionalComputedString("Property a `CAA` record sets – `issue`, `issuewild` or `iodef`. " +
				"Defaults to `issue`."),
			"keytag":      optionalComputedInt64("Key tag of the DNSKEY a `DS` record points at."),
			"algorithm":   optionalComputedInt64("DNSSEC algorithm number of the key a `DS` record points at."),
			"digest_type": optionalComputedInt64("Digest type of a `DS` record, for example `2` for SHA-256."),
			"usage":       optionalComputedInt64("Certificate usage field of a `TLSA` record."),
			"selector": optionalComputedInt64("Selector field of a `TLSA` record: `0` matches the full " +
				"certificate, `1` the subject public key."),
			"matching_type": optionalComputedInt64("Matching type of a `TLSA` record, for example `1` for SHA-256."),
		},
	}
}

// Type-specific record attributes are Optional+Computed: the user may set them,
// but when omitted the API fills in a default (e.g. MX priority 10) that the
// provider reads back from the response "fields" object. UseStateForUnknown keeps
// the stored value stable across plans so unchanged records don't show drift.
func optionalComputedInt64(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func optionalComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func (r *RecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*api.Client)
}

func (r *RecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := r.buildCreateRequest(&data)
	record, err := r.client.CreateRecord(data.ZoneID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create record", err.Error())
		return
	}

	r.recordToModel(record, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.client.GetRecord(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		// A record that is gone is not an error: dropping it from state lets
		// Terraform plan its recreation. Failing here instead left the whole
		// configuration unplannable – and undestroyable – until the user ran
		// `terraform state rm` by hand.
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read record", err.Error())
		return
	}

	r.recordToModel(record, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The id is content-derived, so the planned value is unknown whenever the
	// content changes. The record to address is the one currently in state.
	var state RecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	recordID := state.ID.ValueString()

	content := data.Content.ValueString()
	ttl := int(data.TTL.ValueInt64())
	updateReq := api.UpdateRecordRequest{
		Content: &content,
		TTL:     &ttl,
	}
	if !data.Name.IsNull() {
		name := data.Name.ValueString()
		updateReq.Name = &name
	}
	updateReq.Priority = int64PtrIfSet(data.Priority)
	updateReq.Weight = int64PtrIfSet(data.Weight)
	updateReq.Port = int64PtrIfSet(data.Port)
	updateReq.Flags = int64PtrIfSet(data.Flags)
	updateReq.KeyTag = int64PtrIfSet(data.KeyTag)
	updateReq.Algorithm = int64PtrIfSet(data.Algorithm)
	updateReq.DigestType = int64PtrIfSet(data.DigestType)
	updateReq.Usage = int64PtrIfSet(data.Usage)
	updateReq.Selector = int64PtrIfSet(data.Selector)
	updateReq.MatchingType = int64PtrIfSet(data.MatchingType)
	if !data.Tag.IsNull() && !data.Tag.IsUnknown() {
		tag := data.Tag.ValueString()
		updateReq.Tag = &tag
	}

	record, err := r.client.UpdateRecord(data.ZoneID.ValueString(), recordID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update record", err.Error())
		return
	}

	r.recordToModel(record, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecordResourceModel
	resp.Diagnostics.Append(req.State.Get(context.Background(), &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRecord(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete record", err.Error())
	}
}

func (r *RecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: zone_id/record_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Use format: <zone_id>/<record_id>")
		return
	}

	var data RecordResourceModel
	data.ZoneID = types.StringValue(parts[0])
	data.ID = types.StringValue(parts[1])

	record, err := r.client.GetRecord(parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to read record", err.Error())
		return
	}

	r.recordToModel(record, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) buildCreateRequest(data *RecordResourceModel) api.CreateRecordRequest {
	req := api.CreateRecordRequest{
		Name:    data.Name.ValueString(),
		Type:    data.Type.ValueString(),
		Content: data.Content.ValueString(),
	}
	if !data.TTL.IsNull() && !data.TTL.IsUnknown() {
		req.TTL = int(data.TTL.ValueInt64())
	}
	req.Priority = int64PtrIfSet(data.Priority)
	req.Weight = int64PtrIfSet(data.Weight)
	req.Port = int64PtrIfSet(data.Port)
	req.Flags = int64PtrIfSet(data.Flags)
	req.KeyTag = int64PtrIfSet(data.KeyTag)
	req.Algorithm = int64PtrIfSet(data.Algorithm)
	req.DigestType = int64PtrIfSet(data.DigestType)
	req.Usage = int64PtrIfSet(data.Usage)
	req.Selector = int64PtrIfSet(data.Selector)
	req.MatchingType = int64PtrIfSet(data.MatchingType)
	if !data.Tag.IsNull() && !data.Tag.IsUnknown() {
		req.Tag = data.Tag.ValueString()
	}
	return req
}

// int64PtrIfSet returns a pointer to the int value only when the attribute is
// known and non-null (a Computed+Optional attribute the user omitted is Unknown
// on create, in which case nothing should be sent).
func int64PtrIfSet(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

// recordToModel maps an API record back into the Terraform state model. The API
// response carries type-specific values inside record.Fields (never as top-level
// keys), and Content holds the full record value; this reconstructs the discrete
// attributes and the primary content value to match the user's configuration.
func (r *RecordResource) recordToModel(record *api.Record, data *RecordResourceModel) {
	data.ID = types.StringValue(record.ID)
	data.Name = types.StringValue(preserveConfigured(data.Name.ValueString(), record.Name))
	data.Type = types.StringValue(record.Type)
	data.TTL = types.Int64Value(int64(record.TTL))

	f := record.Fields

	// Default every type-specific attribute to null; the per-type block below
	// populates the ones that apply, so no Computed attribute is left unknown.
	data.Priority = types.Int64Null()
	data.Weight = types.Int64Null()
	data.Port = types.Int64Null()
	data.Flags = types.Int64Null()
	data.Tag = types.StringNull()
	data.KeyTag = types.Int64Null()
	data.Algorithm = types.Int64Null()
	data.DigestType = types.Int64Null()
	data.Usage = types.Int64Null()
	data.Selector = types.Int64Null()
	data.MatchingType = types.Int64Null()

	setInt := func(dst *types.Int64, key string) {
		if v, ok := fieldInt(f, key); ok {
			*dst = types.Int64Value(int64(v))
		}
	}

	// content holds the primary value only (no priority/flags prefix, no trailing
	// dot), matching what the user supplies in configuration.
	content := strings.TrimSuffix(record.Content, ".")
	if v, ok := fieldStr(f, primaryFieldKey(record.Type)); ok {
		content = strings.TrimSuffix(v, ".")
	}

	switch record.Type {
	case "MX":
		setInt(&data.Priority, "priority")
	case "SRV":
		setInt(&data.Priority, "priority")
		setInt(&data.Weight, "weight")
		setInt(&data.Port, "port")
	case "CAA":
		setInt(&data.Flags, "flags")
		if v, ok := fieldStr(f, "tag"); ok {
			data.Tag = types.StringValue(v)
		}
	case "DS":
		setInt(&data.KeyTag, "keytag")
		setInt(&data.Algorithm, "algorithm")
		setInt(&data.DigestType, "digest_type")
	case "TLSA":
		setInt(&data.Usage, "usage")
		setInt(&data.Selector, "selector")
		setInt(&data.MatchingType, "matching_type")
	}

	// The backend lowercases hex rdata (DS and TLSA digests). Writing its casing
	// back over a configuration that used upper case fails the apply with
	// "inconsistent result", so the configured spelling is kept when the two
	// differ only in case – the record itself is identical either way.
	//
	// The same applies to the trailing dot stripped above: a hostname target is
	// the same name with or without it, and writing the stripped form back over
	// a configuration that spelled the FQDN out ("mail.example.com.", which is
	// what the API itself returns) failed every apply.
	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		if planned := data.Content.ValueString(); planned != content && sameName(planned, content) {
			content = planned
		}
	}

	data.Content = types.StringValue(content)
}

// primaryFieldKey returns the key under record.Fields that holds the primary
// value mapped to the "content" attribute for a given record type.
func primaryFieldKey(recordType string) string {
	switch recordType {
	case "A", "AAAA":
		return "address"
	case "CNAME", "NS", "PTR", "ALIAS", "DNAME":
		return "hostname"
	case "MX":
		return "host"
	case "SRV":
		return "target"
	case "CAA", "TXT":
		return "value"
	case "DS":
		return "digest"
	case "TLSA":
		return "certificate"
	default:
		return "content"
	}
}

func fieldInt(f map[string]interface{}, key string) (int, bool) {
	if f == nil {
		return 0, false
	}
	switch n := f[key].(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func fieldStr(f map[string]interface{}, key string) (string, bool) {
	if f == nil {
		return "", false
	}
	if v, ok := f[key].(string); ok {
		return v, true
	}
	return "", false
}
