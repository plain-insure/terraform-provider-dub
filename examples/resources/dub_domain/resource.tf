resource "dub_domain" "example" {
  slug          = "go.example.com"
  archived      = false
  expired_url   = "https://example.com/expired"
  not_found_url = "https://example.com/not-found"
}
