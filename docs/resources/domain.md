---
page_title: "dub_domain Resource - terraform-provider-dub"
subcategory: ""
description: |-
  Manages a domain on a Dub workspace.
---

# dub_domain (Resource)

Manages a domain on a Dub workspace. See the
[Dub domains API reference](https://dub.co/docs/api-reference/endpoint/domains)
for details.

## Example Usage

```terraform
resource "dub_domain" "example" {
  slug          = "go.example.com"
  archived      = false
  expired_url   = "https://example.com/expired"
  not_found_url = "https://example.com/not-found"
}
```

## Schema

### Required

- `slug` (String) The domain name (e.g. `acme.com`). Cannot be changed after creation.

### Optional

- `archived` (Boolean) Whether the domain is archived. Defaults to `false`.
- `expired_url` (String) The URL to redirect to when a link under this domain has expired.
- `logo` (String) The URL of the logo to use as the default OG image for the domain's links (available on Enterprise plans).
- `not_found_url` (String) The URL to redirect to when a link under this domain does not exist.
- `placeholder` (String) Provide context to your teammates in the link creation modal by showing them an example link to be used as a placeholder.

### Read-Only

- `created_at` (String) The date the domain was created.
- `id` (String) The unique ID of the domain.
- `links_count` (Number) The number of links associated with the domain.
- `primary` (Boolean) Whether the domain is the primary domain for the workspace.
- `updated_at` (String) The date the domain was last updated.
- `verified` (Boolean) Whether the domain has been verified to be added to the workspace.

## Import

Domains can be imported using their slug:

```shell
terraform import dub_domain.example go.example.com
```
