# Terraform Provider for Dub

This is a [Terraform](https://www.terraform.io) provider for managing
[Dub](https://dub.co) resources. It is generated/maintained using
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework)
and modeled after the [Dub API](https://dub.co/docs/api-reference)
([OpenAPI spec](https://spec.speakeasy.com/dub/dub/dub-with-code-samples)) via
the [OpenAPI Provider Code Generator](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)
workflow.

Currently, only **domain management** is implemented:

- `dub_domain` (resource) — create, read, update, delete, and import domains
  on a Dub workspace.
- `dub_domain` (data source) — look up a single domain by slug.
- `dub_domains` (data source) — list/search domains on a workspace.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.24 (to build the provider plugin)
- A [Dub API key](https://dub.co/docs/api-reference/tokens)

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
