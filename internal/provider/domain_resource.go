package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dubclient "github.com/plain-insure/terraform-provider-dub/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

// NewDomainResource is a helper function to simplify the provider implementation.
func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

// DomainResource defines the resource implementation for a Dub domain.
type DomainResource struct {
	client *dubclient.Client
}

// DomainResourceModel describes the resource data model.
type DomainResourceModel struct {
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

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a domain on a Dub workspace. See https://dub.co/docs/api-reference/endpoint/domains for details.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique ID of the domain.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "The domain name (e.g. `acme.com`). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"placeholder": schema.StringAttribute{
				Description: "Provide context to your teammates in the link creation modal by showing them an example link to be used as a placeholder.",
				Optional:    true,
				Computed:    true,
			},
			"expired_url": schema.StringAttribute{
				Description: "The URL to redirect to when a link under this domain has expired.",
				Optional:    true,
				Computed:    true,
			},
			"not_found_url": schema.StringAttribute{
				Description: "The URL to redirect to when a link under this domain does not exist.",
				Optional:    true,
				Computed:    true,
			},
			"logo": schema.StringAttribute{
				Description: "The URL of the logo to use as the default OG image for the domain's links (available on Enterprise plans).",
				Optional:    true,
				Computed:    true,
			},
			"links_count": schema.Int64Attribute{
				Description: "The number of links associated with the domain.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the domain was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The date the domain was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dubclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := dubclient.CreateDomainRequest{
		Slug:        plan.Slug.ValueString(),
		ExpiredURL:  stringPointerOrNil(plan.ExpiredURL),
		NotFoundURL: stringPointerOrNil(plan.NotFoundURL),
		Placeholder: stringPointerOrNil(plan.Placeholder),
		Logo:        stringPointerOrNil(plan.Logo),
	}
	if !plan.Archived.IsNull() && !plan.Archived.IsUnknown() {
		archived := plan.Archived.ValueBool()
		in.Archived = &archived
	}

	domain, err := r.client.CreateDomain(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	plan = domainToModel(domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.Slug.ValueString())
	if err != nil {
		if dubclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	state = domainToModel(domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := dubclient.UpdateDomainRequest{
		ExpiredURL:  stringPointerOrNil(plan.ExpiredURL),
		NotFoundURL: stringPointerOrNil(plan.NotFoundURL),
		Placeholder: stringPointerOrNil(plan.Placeholder),
		Logo:        stringPointerOrNil(plan.Logo),
	}
	if !plan.Archived.IsNull() && !plan.Archived.IsUnknown() {
		archived := plan.Archived.ValueBool()
		in.Archived = &archived
	}

	domain, err := r.client.UpdateDomain(ctx, state.Slug.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	plan = domainToModel(domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDomain(ctx, state.Slug.ValueString()); err != nil {
		if dubclient.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

// domainToModel converts an API domain into the Terraform resource model.
func domainToModel(d *dubclient.Domain) DomainResourceModel {
	return DomainResourceModel{
		ID:          types.StringValue(d.ID),
		Slug:        types.StringValue(d.Slug),
		Verified:    types.BoolValue(d.Verified),
		Primary:     types.BoolValue(d.Primary),
		Archived:    types.BoolValue(d.Archived),
		Placeholder: types.StringValue(d.Placeholder),
		ExpiredURL:  types.StringValue(d.ExpiredURL),
		NotFoundURL: types.StringValue(d.NotFoundURL),
		Logo:        types.StringValue(d.Logo),
		LinksCount:  types.Int64Value(int64(d.LinksCount)),
		CreatedAt:   types.StringValue(d.CreatedAt),
		UpdatedAt:   types.StringValue(d.UpdatedAt),
	}
}

// stringPointerOrNil returns a pointer to the string value if it is known
// and non-null, otherwise nil.
func stringPointerOrNil(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	val := v.ValueString()
	return &val
}
