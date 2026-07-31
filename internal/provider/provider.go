package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexdns/terraform-provider-nexdns/internal/api"
)

var _ provider.Provider = &NexdnsProvider{}

type NexdnsProvider struct {
	version string
}

type NexdnsProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	APIUrl   types.String `tfsdk:"api_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NexdnsProvider{version: version}
	}
}

func (p *NexdnsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nexdns"
	resp.Version = p.version
}

func (p *NexdnsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage DNS zones, records and DNSSEC signing on the NexDNS platform through its REST API.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token used for every call. Create one under " +
					"[API keys](https://nexdns.tech/settings/api-keys) on an account whose plan includes API access, " +
					"and grant it the scopes the configuration needs: `zones.read` and `zones.write` for zones and " +
					"DNSSEC, `records.read` and `records.write` for records. May also be supplied through the " +
					"`NEXDNS_API_TOKEN` environment variable, which keeps the token out of the configuration; a " +
					"value set here wins over the variable.",
				Optional:  true,
				Sensitive: true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the REST API. Defaults to `https://api.nexdns.tech/v1`. " +
					"May also be supplied through the `NEXDNS_API_URL` environment variable; a value set here wins " +
					"over the variable.",
				Optional: true,
			},
		},
	}
}

func (p *NexdnsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config NexdnsProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("NEXDNS_API_TOKEN")
	if !config.APIToken.IsNull() {
		token = config.APIToken.ValueString()
	}

	apiUrl := "https://api.nexdns.tech/v1"
	if v := os.Getenv("NEXDNS_API_URL"); v != "" {
		apiUrl = v
	}
	if !config.APIUrl.IsNull() {
		apiUrl = config.APIUrl.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"Set api_token in provider config or NEXDNS_API_TOKEN environment variable.",
		)
		return
	}

	client := api.NewClient(apiUrl, token)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NexdnsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewRecordResource,
		NewDNSSECResource,
	}
}

func (p *NexdnsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZoneDataSource,
	}
}
