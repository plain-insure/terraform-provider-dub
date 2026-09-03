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
