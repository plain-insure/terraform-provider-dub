package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dubclient "github.com/plain-insure/terraform-provider-dub/internal/client"
)

var (
	_ datasource.DataSource              = &DomainDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainDataSource{}
)

// NewDomainDataSource is a helper function to simplify the provider implementation.
func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

// DomainDataSource defines the data source implementation for a single Dub domain.
type DomainDataSource struct {
	client *dubclient.Client
}

func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an existing domain on a Dub workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique ID of the domain.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The domain name (e.g. `acme.com`).",
				Required:    true,
			},
			"verified": schema.BoolAttribute{
				Description: "Whether the domain has been verified to be added to the workspace.",
				Computed:    true,
			},
			"primary": schema.BoolAttribute{
				Description: "Whether the domain is the primary domain for the workspace.",
				Computed:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the domain is archived.",
				Computed:    true,
			},
			"placeholder": schema.StringAttribute{
				Description: "Example link shown as a placeholder in the link creation modal.",
				Computed:    true,
			},
			"expired_url": schema.StringAttribute{
				Description: "The URL to redirect to when a link under this domain has expired.",
				Computed:    true,
			},
			"not_found_url": schema.StringAttribute{
				Description: "The URL to redirect to when a link under this domain does not exist.",
				Computed:    true,
			},
			"logo": schema.StringAttribute{
				Description: "The URL of the logo to use as the default OG image for the domain's links.",
				Computed:    true,
			},
			"links_count": schema.Int64Attribute{
				Description: "The number of links associated with the domain.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the domain was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The date the domain was last updated.",
				Computed:    true,
			},
		},
	}
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dubclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, config.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	state := domainToModel(domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// DomainSummaryModel is the per-item model used by the dub_domains list data source.
type DomainSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Verified    types.Bool   `tfsdk:"verified"`
	Primary     types.Bool   `tfsdk:"primary"`
	Archived    types.Bool   `tfsdk:"archived"`
	Placeholder types.String `tfsdk:"placeholder"`
	ExpiredURL  types.String `tfsdk:"expired_url"`
	NotFoundURL types.String `tfsdk:"not_found_url"`
	Logo        types.String `tfsdk:"logo"`
	LinksCount  types.Int64  `tfsdk:"links_count"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// DomainsDataSourceModel describes the dub_domains list data source data model.
type DomainsDataSourceModel struct {
	Search   types.String         `tfsdk:"search"`
	Archived types.Bool           `tfsdk:"archived"`
	Domains  []DomainSummaryModel `tfsdk:"domains"`
}

var (
	_ datasource.DataSource              = &DomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainsDataSource{}
)

// NewDomainsDataSource is a helper function to simplify the provider implementation.
func NewDomainsDataSource() datasource.DataSource {
	return &DomainsDataSource{}
}

// DomainsDataSource defines the data source implementation for listing Dub domains.
type DomainsDataSource struct {
	client *dubclient.Client
}

func (d *DomainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *DomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of domains on a Dub workspace.",
		Attributes: map[string]schema.Attribute{
			"search": schema.StringAttribute{
				Description: "Only return domains whose slug contains this search term.",
				Optional:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Only return domains with the given archived status.",
				Optional:    true,
			},
			"domains": schema.ListNestedAttribute{
				Description: "The list of domains matching the given filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique ID of the domain.",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The domain name (e.g. `acme.com`).",
							Computed:    true,
						},
						"verified": schema.BoolAttribute{
							Description: "Whether the domain has been verified to be added to the workspace.",
							Computed:    true,
						},
						"primary": schema.BoolAttribute{
							Description: "Whether the domain is the primary domain for the workspace.",
							Computed:    true,
						},
						"archived": schema.BoolAttribute{
							Description: "Whether the domain is archived.",
							Computed:    true,
						},
						"placeholder": schema.StringAttribute{
							Description: "Example link shown as a placeholder in the link creation modal.",
							Computed:    true,
						},
						"expired_url": schema.StringAttribute{
							Description: "The URL to redirect to when a link under this domain has expired.",
							Computed:    true,
						},
						"not_found_url": schema.StringAttribute{
							Description: "The URL to redirect to when a link under this domain does not exist.",
							Computed:    true,
						},
						"logo": schema.StringAttribute{
							Description: "The URL of the logo to use as the default OG image for the domain's links.",
							Computed:    true,
						},
						"links_count": schema.Int64Attribute{
							Description: "The number of links associated with the domain.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The date the domain was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The date the domain was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dubclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *DomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := dubclient.ListDomainsParams{}
	if !config.Search.IsNull() {
		params.Search = config.Search.ValueString()
	}
	if !config.Archived.IsNull() {
		archived := config.Archived.ValueBool()
		params.Archived = &archived
	}

	domains, err := d.client.ListDomains(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error listing domains", err.Error())
		return
	}

	items := make([]DomainSummaryModel, 0, len(domains))
	for i := range domains {
		m := domainToModel(&domains[i])
		items = append(items, DomainSummaryModel{
			ID:          m.ID,
			Slug:        m.Slug,
			Verified:    m.Verified,
			Primary:     m.Primary,
			Archived:    m.Archived,
			Placeholder: m.Placeholder,
			ExpiredURL:  m.ExpiredURL,
			NotFoundURL: m.NotFoundURL,
			Logo:        m.Logo,
			LinksCount:  m.LinksCount,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}
	config.Domains = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
