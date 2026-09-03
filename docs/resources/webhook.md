---
page_title: "dub_webhook Resource - terraform-provider-dub"
subcategory: ""
description: |-
  Manages a webhook on a Dub workspace.
---

# dub_webhook (Resource)

Manages a webhook on a Dub workspace.

Webhook management uses Dub's undocumented, unversioned app API rather than
the public REST API. This endpoint can change without notice. The provider's
acceptance test exercises the live endpoint to detect API drift early.

The workspace must be on the Business, Advanced, or Enterprise plan. The API
key needs both `webhooks.read` and `webhooks.write` scopes; `apis.all` alone is
not sufficient. Set `workspace_id` in the provider configuration or
`DUB_WORKSPACE_ID` so requests explicitly target the intended workspace.

## API Key Setup

Dub's regular API key UI does not expose the webhook scopes. Create or modify
the key in the Dub UI, capture that request in the browser Network panel, and
replay it with the required scopes. The replay uses your authenticated browser
session automatically; never copy or expose its `Cookie` header.

1. Open the create or edit API key screen for the target Dub workspace.
2. In Chrome or Firefox Developer Tools, open **Network** and create or save a
  key to capture the request.
3. Right-click the request and select **Edit and Resend**.
4. Update the request JSON and resend it. A Terraform key needs both webhook
  permissions:

  ```json
  {"name":"Terraform","scopes":["apis.all","webhooks.read","webhooks.write"],"isMachine":true}
  ```

  To diagnose a read-only `403` that requests `webhooks.read`, add that scope
  explicitly, even when the key already has `apis.all`:

  ```json
  {"name":"Claude TEST","scopes":["apis.all","webhooks.read"],"isMachine":false}
  ```

5. Use the replayed response's API key as `DUB_API_KEY` or the provider's
  sensitive `api_key` value.

## Example Usage

```terraform
resource "dub_webhook" "link_events" {
  name     = "Link event receiver"
  url      = "https://example.com/dub/webhooks"
  triggers = ["link.created", "link.updated"]
}

resource "dub_webhook" "clicked_links" {
  name       = "Selected link clicks"
  url        = "https://example.com/dub/clicks"
  triggers   = ["link.clicked"]
  link_scope = "links"
  link_ids   = ["lnk_123", "lnk_456"]
}
```

## Schema

### Required

- `name` (String) Name of the webhook. Must be 1 to 40 characters.
- `triggers` (Set of String) Events that invoke the webhook. Must contain at
  least one supported event.
- `url` (String) Absolute URL to receive webhook events.

### Optional

- `folder_ids` (Set of String) Folder IDs included when `link_scope` is
  `folders`. At most 100 IDs. This is write-only and is not refreshed from the
  API.
- `link_ids` (Set of String) Link IDs included when `link_scope` is `links`.
  At most 1000 IDs. This is write-only and is not refreshed from the API.
- `link_scope` (String) Scope for `link.clicked`: `links` or `folders`.

When `triggers` includes `link.clicked`, `link_scope` is required. A `links`
scope requires non-empty `link_ids`; a `folders` scope requires non-empty
`folder_ids`. For every other trigger set, omit `link_scope`.

Because Dub does not return `link_ids` or `folder_ids`, the provider preserves
their configured state during refresh. Out-of-band changes to this scope are
not detected by `terraform plan`.

### Read-Only

- `disabled_at` (String) When Dub disabled the webhook after repeated delivery
  failures.
- `id` (String) Unique webhook ID.
- `installation_id` (String) ID of the integration installation managing the
  webhook, when applicable. Integration-managed webhooks cannot be imported or
  updated by this provider.
- `secret` (String, Sensitive) Server-generated signing secret.

## Import

Webhooks can be imported using their ID:

```shell
terraform import dub_webhook.example wh_xxxxxxxxxxxxxxxx
```

Integration-managed webhooks, such as those created by Zapier, cannot be
imported for Terraform management. Avoid Zapier hook URLs unless that
integration behavior is intended.