// Package provider implements the Terraform provider for managing Dub
// (https://dub.co) resources. Only domain management is currently
// implemented.
package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dubclient "github.com/plain-insure/terraform-provider-dub/internal/client"
)

// Ensure DubProvider satisfies various provider interfaces.
var _ provider.Provider = &DubProvider{}

// New returns a function that creates a new instance of the Dub provider,
// suitable for use with providerserver.NewProtocol6/5.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DubProvider{version: version}
	}
}

// DubProvider is the provider implementation.
type DubProvider struct {
	// version is set to the provider version at release time.
	version string
}

// DubProviderModel describes the provider-level configuration data.
type DubProviderModel struct {
	APIKey      types.String `tfsdk:"api_key"`
	BaseURL     types.String `tfsdk:"base_url"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func (p *DubProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dub"
	resp.Version = p.version
}

func (p *DubProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Dub provider allows Terraform to manage domains on a Dub (https://dub.co) workspace.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The Dub API key/access token used to authenticate with the Dub API. " +
					"May also be provided via the `DUB_API_KEY` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL of the Dub API. Defaults to `https://api.dub.co`. " +
					"May also be provided via the `DUB_API_URL` environment variable.",
				Optional: true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "The ID (or slug prefixed with `ws_`) of the Dub workspace to manage. " +
					"May also be provided via the `DUB_WORKSPACE_ID` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *DubProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DubProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("DUB_API_KEY")
	if !data.APIKey.IsNull() && data.APIKey.ValueString() != "" {
		apiKey = data.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Dub API Key",
			"The provider requires an API key to authenticate with the Dub API. "+
				"Set the api_key attribute in the provider configuration or use the DUB_API_KEY environment variable.",
		)
		return
	}

	baseURL := os.Getenv("DUB_API_URL")
	if !data.BaseURL.IsNull() && data.BaseURL.ValueString() != "" {
		baseURL = data.BaseURL.ValueString()
	}

	workspaceID := os.Getenv("DUB_WORKSPACE_ID")
	if !data.WorkspaceID.IsNull() && data.WorkspaceID.ValueString() != "" {
		workspaceID = data.WorkspaceID.ValueString()
	}

	client := dubclient.New(baseURL, apiKey, workspaceID, http.DefaultClient)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DubProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
	}
}

func (p *DubProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewDomainsDataSource,
	}
}
