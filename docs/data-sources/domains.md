---
page_title: "dub_domains Data Source - terraform-provider-dub"
subcategory: ""
description: |-
  Retrieves the list of domains on a Dub workspace.
---

# dub_domains (Data Source)

Retrieves the list of domains on a Dub workspace.

## Example Usage

```terraform
data "dub_domains" "example" {
  archived = false
}
```

## Schema

### Optional

- `archived` (Boolean) Only return domains with the given archived status.
- `search` (String) Only return domains whose slug contains this search term.

### Read-Only

- `domains` (Attributes List) The list of domains matching the given filters. (see [below for nested schema](#nestedatt--domains))

<a id="nestedatt--domains"></a>
### Nested Schema for `domains`

Read-Only:

- `archived` (Boolean) Whether the domain is archived.
- `created_at` (String) The date the domain was created.
- `expired_url` (String) The URL to redirect to when a link under this domain has expired.
- `id` (String) The unique ID of the domain.
- `links_count` (Number) The number of links associated with the domain.
- `logo` (String) The URL of the logo to use as the default OG image for the domain's links.
- `not_found_url` (String) The URL to redirect to when a link under this domain does not exist.
- `placeholder` (String) Example link shown as a placeholder in the link creation modal.
- `primary` (Boolean) Whether the domain is the primary domain for the workspace.
- `slug` (String) The domain name (e.g. `acme.com`).
- `updated_at` (String) The date the domain was last updated.
- `verified` (Boolean) Whether the domain has been verified to be added to the workspace.
