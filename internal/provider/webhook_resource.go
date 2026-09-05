package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dubclient "github.com/plain-insure/terraform-provider-dub/internal/client"
)

const maxWebhookFolders = 100

var webhookTriggers = map[string]struct{}{
	"link.created":                  {},
	"link.updated":                  {},
	"link.deleted":                  {},
	"link.clicked":                  {},
	"lead.created":                  {},
	"sale.created":                  {},
	"partner.application_submitted": {},
	"partner.enrolled":              {},
	"partner.merged":                {},
	"commission.created":            {},
	"bounty.created":                {},
	"bounty.updated":                {},
	"payout.confirmed":              {},
	"discount_code.created":         {},
	"discount_code.deleted":         {},
}

var (
	_ resource.Resource                     = &WebhookResource{}
	_ resource.ResourceWithConfigure        = &WebhookResource{}
	_ resource.ResourceWithConfigValidators = &WebhookResource{}
	_ resource.ResourceWithImportState      = &WebhookResource{}
)

// NewWebhookResource creates the dub_webhook resource.
func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

// WebhookResource manages a Dub webhook through the Dub app API.
type WebhookResource struct {
	client *dubclient.WebhookClient
}

// WebhookResourceModel describes the resource data model.
type WebhookResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	URL            types.String `tfsdk:"url"`
	Triggers       types.Set    `tfsdk:"triggers"`
	LinkScope      types.String `tfsdk:"link_scope"`
	LinkIDs        types.Set    `tfsdk:"link_ids"`
	FolderIDs      types.Set    `tfsdk:"folder_ids"`
	Secret         types.String `tfsdk:"secret"`
	DisabledAt     types.String `tfsdk:"disabled_at"`
	InstallationID types.String `tfsdk:"installation_id"`
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a webhook on a Dub workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique ID of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A name for the webhook. Must be between 1 and 40 characters.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "The HTTPS endpoint to receive webhook events.",
				Required:    true,
			},
			"triggers": schema.SetAttribute{
				Description: "The events that invoke the webhook.",
				Required:    true,
				ElementType: types.StringType,
			},
			"link_scope": schema.StringAttribute{
				Description: "Scope for link.clicked events: links or folders.",
				Optional:    true,
			},
			"link_ids": schema.SetAttribute{
				Description: "Link IDs included when link_scope is links. This write-only setting is not refreshed from the API.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"folder_ids": schema.SetAttribute{
				Description: "Folder IDs included when link_scope is folders. This write-only setting is not refreshed from the API.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"secret": schema.StringAttribute{
				Description: "The server-generated signing secret for the webhook.",
				Computed:    true,
				Sensitive:   true,
			},
			"disabled_at": schema.StringAttribute{
				Description: "When the server disabled the webhook after repeated delivery failures.",
				Computed:    true,
			},
			"installation_id": schema.StringAttribute{
				Description: "The integration installation that manages this webhook, when applicable.",
				Computed:    true,
			},
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.Webhooks
}

func (r *WebhookResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{webhookConfigValidator{}}
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.CreateWebhook(ctx, webhookRequest(plan))
	if err != nil {
		addWebhookError(&resp.Diagnostics, "Error creating webhook", err)
		return
	}

	plan = webhookToModel(webhook, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if err != nil {
		if dubclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addWebhookError(&resp.Diagnostics, "Error reading webhook", err)
		return
	}
	if webhook.InstallationID != nil && state.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Webhook is managed by an integration",
			"This webhook is managed by a Dub integration and cannot be imported for Terraform management.",
		)
		return
	}

	state = webhookToModel(webhook, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.InstallationID.IsNull() && !state.InstallationID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Webhook is managed by an integration",
			"This webhook is managed by a Dub integration and cannot be updated through Terraform.",
		)
		return
	}

	webhook, err := r.client.UpdateWebhook(ctx, state.ID.ValueString(), webhookRequest(plan))
	if err != nil {
		addWebhookError(&resp.Diagnostics, "Error updating webhook", err)
		return
	}

	plan = webhookToModel(webhook, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWebhook(ctx, state.ID.ValueString()); err != nil && !dubclient.IsNotFound(err) {
		addWebhookError(&resp.Diagnostics, "Error deleting webhook", err)
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func webhookRequest(model WebhookResourceModel) dubclient.WebhookRequest {
	return dubclient.WebhookRequest{
		Name:      model.Name.ValueString(),
		URL:       model.URL.ValueString(),
		Triggers:  stringsFromSet(model.Triggers),
		LinkScope: stringPointerOrNil(model.LinkScope),
		LinkIDs:   stringsFromSet(model.LinkIDs),
		FolderIDs: stringsFromSet(model.FolderIDs),
	}
}

func webhookToModel(webhook *dubclient.Webhook, current WebhookResourceModel) WebhookResourceModel {
	current.ID = types.StringValue(webhook.ID)
	current.Name = types.StringValue(webhook.Name)
	current.URL = types.StringValue(webhook.URL)
	current.Triggers = types.SetValueMust(types.StringType, stringValues(webhook.Triggers))
	current.LinkScope = nullableString(webhook.LinkScope)
	current.Secret = types.StringValue(webhook.Secret)
	current.DisabledAt = nullableString(webhook.DisabledAt)
	current.InstallationID = nullableString(webhook.InstallationID)
	return current
}

func stringsFromSet(set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	values := make([]string, 0, len(set.Elements()))
	for _, element := range set.Elements() {
		stringValue, ok := element.(types.String)
		if ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			values = append(values, stringValue.ValueString())
		}
	}
	return values
}

func stringValues(values []string) []attr.Value {
	result := make([]attr.Value, len(values))
	for index, value := range values {
		result[index] = types.StringValue(value)
	}
	return result
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func addWebhookError(diagnostics *diag.Diagnostics, summary string, err error) {
	if dubclient.IsForbidden(err) {
		diagnostics.AddError(
			"Webhook access denied",
			"Webhook management requires a Business, Advanced, or Enterprise workspace and an API key with webhooks.read and webhooks.write scopes.",
		)
		return
	}
	diagnostics.AddError(summary, err.Error())
}

type webhookConfigValidator struct{}

func (webhookConfigValidator) Description(context.Context) string {
	return "Validates webhook event and link scope configuration."
}

func (webhookConfigValidator) MarkdownDescription(ctx context.Context) string {
	return webhookConfigValidator{}.Description(ctx)
}

func (webhookConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config WebhookResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Name.IsUnknown() && !config.Name.IsNull() {
		nameLength := len(config.Name.ValueString())
		if nameLength < 1 || nameLength > 40 {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid webhook name", "name must be between 1 and 40 characters.")
		}
	}
	if !config.URL.IsUnknown() && !config.URL.IsNull() {
		parsedURL, err := url.ParseRequestURI(config.URL.ValueString())
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid webhook URL", "url must be a valid absolute URL.")
		}
	}

	if config.Triggers.IsUnknown() || config.Triggers.IsNull() {
		return
	}
	triggers := stringsFromSet(config.Triggers)
	if len(triggers) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Webhook requires a trigger", "triggers must contain at least one event.")
	}

	clicked := false
	for _, trigger := range triggers {
		if _, ok := webhookTriggers[trigger]; !ok {
			resp.Diagnostics.AddAttributeError(path.Root("triggers"), "Invalid webhook trigger", fmt.Sprintf("%q is not a supported webhook trigger.", trigger))
		}
		if trigger == "link.clicked" {
			clicked = true
		}
	}

	if !config.LinkIDs.IsUnknown() && len(stringsFromSet(config.LinkIDs)) > 1000 {
		resp.Diagnostics.AddAttributeError(path.Root("link_ids"), "Too many webhook links", "link_ids can contain at most 1000 IDs.")
	}
	if !config.FolderIDs.IsUnknown() && len(stringsFromSet(config.FolderIDs)) > maxWebhookFolders {
		resp.Diagnostics.AddAttributeError(path.Root("folder_ids"), "Too many webhook folders", "folder_ids can contain at most 100 IDs.")
	}

	if !clicked {
		if !config.LinkScope.IsNull() && !config.LinkScope.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root("link_scope"), "Unexpected link scope", "link_scope can only be set when triggers includes link.clicked.")
		}
		return
	}
	if config.LinkScope.IsUnknown() {
		return
	}
	if config.LinkScope.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("link_scope"), "Missing link scope", "link_scope is required when triggers includes link.clicked.")
		return
	}

	scope := config.LinkScope.ValueString()
	switch scope {
	case "links":
		if config.LinkIDs.IsNull() || (!config.LinkIDs.IsUnknown() && len(stringsFromSet(config.LinkIDs)) == 0) {
			resp.Diagnostics.AddAttributeError(path.Root("link_ids"), "Missing webhook links", "link_ids must contain at least one ID when link_scope is links.")
		}
	case "folders":
		if config.FolderIDs.IsNull() || (!config.FolderIDs.IsUnknown() && len(stringsFromSet(config.FolderIDs)) == 0) {
			resp.Diagnostics.AddAttributeError(path.Root("folder_ids"), "Missing webhook folders", "folder_ids must contain at least one ID when link_scope is folders.")
		}
	default:
		resp.Diagnostics.AddAttributeError(path.Root("link_scope"), "Invalid link scope", "link_scope must be links or folders.")
	}
}
