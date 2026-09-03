---
page_title: "dub Provider"
subcategory: ""
description: |-
  The Dub provider allows Terraform to manage domains on a Dub (https://dub.co) workspace.
---

# dub Provider

The Dub provider allows Terraform to manage domains on a [Dub](https://dub.co)
workspace. It currently supports managing domains only.

## Example Usage

```terraform
terraform {
  required_providers {
    dub = {
      source = "plain-insure/dub"
    }
  }
}

provider "dub" {
  # The api_key can also be sourced from the DUB_API_KEY environment variable.
  api_key = var.dub_api_key

  # The workspace_id can also be sourced from the DUB_WORKSPACE_ID environment variable.
  workspace_id = "ws_xxxxxxxxxxxxxxxxxxxxxxxx"
}
```

## Schema

### Optional

- `api_key` (String, Sensitive) The Dub API key/access token used to authenticate with the Dub API. May also be provided via the `DUB_API_KEY` environment variable.
- `base_url` (String) The base URL of the Dub API. Defaults to `https://api.dub.co`. May also be provided via the `DUB_API_URL` environment variable.
- `workspace_id` (String) The ID (or slug prefixed with `ws_`) of the Dub workspace to manage. May also be provided via the `DUB_WORKSPACE_ID` environment variable.
