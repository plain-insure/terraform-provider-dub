package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	dubclient "github.com/plain-insure/terraform-provider-dub/internal/client"
)

func TestProvider_SchemaValid(t *testing.T) {
	ctx := context.Background()

	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["api_key"]; !ok {
		t.Fatal("expected api_key attribute in provider schema")
	}
	if _, ok := resp.Schema.Attributes["base_url"]; !ok {
		t.Fatal("expected base_url attribute in provider schema")
	}
	if _, ok := resp.Schema.Attributes["workspace_id"]; !ok {
		t.Fatal("expected workspace_id attribute in provider schema")
	}
}

func TestProvider_Metadata(t *testing.T) {
	ctx := context.Background()

	p := New("1.2.3")()
	resp := &provider.MetadataResponse{}
	p.Metadata(ctx, provider.MetadataRequest{}, resp)

	if resp.TypeName != "dub" {
		t.Fatalf("expected type name 'dub', got %q", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("expected version '1.2.3', got %q", resp.Version)
	}
}

func TestWebhookResource_ConfigureWithoutWorkspaceID(t *testing.T) {
	webhookResource := &WebhookResource{}
	resp := &resource.ConfigureResponse{}

	webhookResource.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: dubclient.New("", "test-api-key", "", nil),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}
