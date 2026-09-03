---
page_title: "dub_domain Data Source - terraform-provider-dub"
subcategory: ""
description: |-
  Retrieves information about an existing domain on a Dub workspace.
---

# dub_domain (Data Source)

Retrieves information about an existing domain on a Dub workspace.

## Example Usage

```terraform
data "dub_domain" "example" {
  slug = "go.example.com"
}
```

## Schema

### Required

- `slug` (String) The domain name (e.g. `acme.com`).

### Read-Only

- `archived` (Boolean) Whether the domain is archived.
- `created_at` (String) The date the domain was created.
- `expired_url` (String) The URL to redirect to when a link under this domain has expired.
- `id` (String) The unique ID of the domain.
- `links_count` (Number) The number of links associated with the domain.
- `logo` (String) The URL of the logo to use as the default OG image for the domain's links.
- `not_found_url` (String) The URL to redirect to when a link under this domain does not exist.
- `placeholder` (String) Example link shown as a placeholder in the link creation modal.
- `primary` (Boolean) Whether the domain is the primary domain for the workspace.
- `updated_at` (String) The date the domain was last updated.
- `verified` (Boolean) Whether the domain has been verified to be added to the workspace.
