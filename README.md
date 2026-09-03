# Terraform Provider for Dub

This is a [Terraform](https://www.terraform.io) provider for managing
[Dub](https://dub.co) resources. It is generated/maintained using
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework)
and modeled after the [Dub API](https://dub.co/docs/api-reference)
([OpenAPI spec](https://spec.speakeasy.com/dub/dub/dub-with-code-samples)) via
the [OpenAPI Provider Code Generator](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)
workflow.

Supported resources and data sources:

- `dub_domain` (resource) — create, read, update, delete, and import domains
  on a Dub workspace.
- `dub_domain` (data source) — look up a single domain by slug.
- `dub_domains` (data source) — list/search domains on a workspace.
- `dub_webhook` (resource) — create, read, update, delete, and import
  webhooks on a Dub workspace.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.24 (to build the provider plugin)
- A [Dub API key](https://dub.co/docs/api-reference/tokens)

## Webhook API Key Permissions

`dub_webhook` uses an undocumented Dub app API. Standard API key scopes,
including `apis.all`, do not grant access to this API. The key must explicitly
include `webhooks.read` to read webhooks and `webhooks.write` to create, update,
or delete them. The workspace must also be on the Business, Advanced, or
Enterprise plan.

When a scope is missing, Dub responds with an error such as:

```json
{"error":{"code":"forbidden","message":"The provided key does not have the required permissions for this endpoint on the workspace 'ws_...'. Having the 'webhooks.read' permission would allow this request to continue."}}
```

Dub's normal API key UI does not expose these scopes. Create or modify the key
in the UI, then use the browser's Network panel to replay the authenticated
request with the required scopes. Browser replay preserves the session cookie;
do not copy, share, or paste a `Cookie` header into a terminal or source file.

1. Sign in to Dub and open the API key create or edit screen for the workspace.
2. Open Developer Tools, select **Network**, and create or save an API key to
  capture its request.
3. In Chrome, right-click the request and select **Edit and Resend**. In
  Firefox, right-click it and select **Edit and Resend**.
4. Change the JSON request body to include the webhook scopes, then resend it.
  For a key that only needs to read webhooks:

  ```json
  {"name":"Claude TEST","scopes":["apis.all","webhooks.read"],"isMachine":false}
  ```

  Terraform manages webhook lifecycle operations, so its key needs both
  permissions:

  ```json
  {"name":"Terraform","scopes":["apis.all","webhooks.read","webhooks.write"],"isMachine":true}
  ```

5. Copy the API key returned by the replayed request and provide it through
  `DUB_API_KEY` or the provider's sensitive `api_key` setting.

## Building the provider

```shell
go build -o terraform-provider-dub
```

## Using the provider

```hcl
terraform {
  required_providers {
    dub = {
      source = "plain-insure/dub"
    }
  }
}

provider "dub" {
  api_key      = var.dub_api_key      # or set DUB_API_KEY
  workspace_id = "ws_xxxxxxxxxxxxxxxx" # or set DUB_WORKSPACE_ID
}

resource "dub_domain" "example" {
  slug          = "go.example.com"
  archived      = false
  expired_url   = "https://example.com/expired"
  not_found_url = "https://example.com/not-found"
}

data "dub_domain" "example" {
  slug = "go.example.com"
}

data "dub_domains" "all" {}

resource "dub_webhook" "example" {
  name     = "Link notifications"
  url      = "https://example.com/dub/webhooks"
  triggers = ["link.created", "link.updated"]
}
```

See [`docs/`](./docs) for full resource/data source reference and
[`examples/`](./examples) for additional usage samples.

## Development

```shell
# Run unit tests
go test ./...

# Run acceptance tests (requires a real Dub API key)
DUB_API_KEY=... TF_ACC=1 go test ./... -run TestAcc -v
```
