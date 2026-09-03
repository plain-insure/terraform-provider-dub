package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/plain-insure/terraform-provider-dub/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"dub": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("DUB_API_KEY") == "" {
		t.Skip("DUB_API_KEY must be set for acceptance tests")
	}
}

// TestAccDomainResource is a full lifecycle acceptance test that exercises
// the dub_domain resource against the real Dub API. It only runs when
// TF_ACC=1 and DUB_API_KEY are set, consistent with terraform-plugin-testing
// conventions.
func TestAccDomainResource(t *testing.T) {
	_ = context.Background()

	slug := fmt.Sprintf("tf-acc-test-%d.example.com", os.Getpid())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig(slug, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("dub_domain.test", "slug", slug),
					resource.TestCheckResourceAttr("dub_domain.test", "archived", "false"),
					resource.TestCheckResourceAttrSet("dub_domain.test", "id"),
				),
			},
			{
				ResourceName:      "dub_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDomainResourceConfig(slug, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("dub_domain.test", "slug", slug),
					resource.TestCheckResourceAttr("dub_domain.test", "archived", "true"),
				),
			},
		},
	})
}

func testAccDomainResourceConfig(slug string, archived bool) string {
	return fmt.Sprintf(`
resource "dub_domain" "test" {
  slug     = %q
  archived = %t
}
`, slug, archived)
}
