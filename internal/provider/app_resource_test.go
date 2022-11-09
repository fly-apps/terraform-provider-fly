package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	providerGraphql "github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestAccApp_basic(t *testing.T) {
	appName := "testApp"
	resourceName := fmt.Sprintf("fly_app.%s", appName)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	ctx := context.Background()
	h := http.Client{Timeout: 60 * time.Second, Transport: &utils.Transport{UnderlyingTransport: http.DefaultTransport, Token: os.Getenv("FLY_API_TOKEN"), Ctx: context.Background()}}
	client := graphql.NewClient("https://api.fly.io/graphql", &h)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "org", getTestOrg()),
					resource.TestCheckResourceAttrSet(resourceName, "orgid"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrWith(resourceName, "id", func(id string) error {
						app, err := providerGraphql.GetApp(ctx, client, id)
						if err != nil {
							t.Fatalf("Error in GetApp for %s: %v", id, err)
						}
						if app == nil {
							t.Fatalf("GetApp for %s returned nil", id)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccApp_secrets(t *testing.T) {
	appName := "testApp"
	resourceName := fmt.Sprintf("fly_app.%s", appName)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	h := http.Client{Timeout: 60 * time.Second, Transport: &utils.Transport{UnderlyingTransport: http.DefaultTransport, Token: os.Getenv("FLY_API_TOKEN"), Ctx: context.Background()}}
	client := graphql.NewClient("https://api.fly.io/graphql", &h)

	var lastDigest string

	testDigestEqualInApi := func(digest string) error {
		r, err := providerGraphql.GetSecrets(context.Background(), client, name)
		if err != nil {
			t.Fatal(err)
		}
		apiDigest := r.App.Secrets[0].Digest
		if digest != apiDigest {
			return errors.New(fmt.Sprintf("Digest %s in resource differs from digest %s from API", digest, apiDigest))
		}
		return nil
	}

	testDigestChanged := func(digest string) error {
		if digest == lastDigest {
			return errors.New(fmt.Sprintf("digest %s did not change even though we changed the secret's value", digest))
		}
		lastDigest = digest
		return nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
	secrets = {
		TEST = {value = "1"}
	}
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "secrets.TEST.value", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.digest"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.created_at"),
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", testDigestEqualInApi),
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", func(digest string) error {
						lastDigest = digest
						return nil
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
	secrets = {
		TEST = {value = "2"}
	}
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "secrets.TEST.value", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.digest"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.created_at"),
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", testDigestChanged),
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", testDigestEqualInApi),
				),
			},

			// Secret drift detection: We change the secret's value using direct API calls outside
			// and verify that the resource is able to detect this and restore the original state
			{
				PreConfig: func() {
					r, err := providerGraphql.GetSecrets(context.Background(), client, name)
					if err != nil {
						t.Fatal(err)
					}
					oldSecret := r.App.Secrets[0]
					sr, err := providerGraphql.SetSecrets(context.Background(), client, providerGraphql.SetSecretsInput{
						AppId: name,
						Secrets: []providerGraphql.SecretInput{{
							Key:   "TEST",
							Value: "3",
						}},
					})
					if err != nil {
						t.Fatal(err)
					}
					r, err = providerGraphql.GetSecrets(context.Background(), client, name)
					newSecret := r.App.Secrets[0]
					if sr.SetSecrets.App.Secrets[0].Digest != newSecret.Digest {
						t.Fatal("fly API SetSecrets returned different digest than subsequent GetSecrets")
					}
					if newSecret.Digest == oldSecret.Digest {
						t.Fatal("fly API SetSecret had no effect")
					}
				},
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
	secrets = {
		TEST = {value = "2"}
	}
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "secrets.TEST.value", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.digest"),
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.created_at"),
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", testDigestEqualInApi),
					func(state *terraform.State) error {
						r, err := providerGraphql.GetSecrets(context.Background(), client, name)
						if err != nil {
							return err
						}
						if len(r.App.Secrets) != 1 {
							return errors.New(fmt.Sprintf("Unexpected number of secrets %d", len(r.App.Secrets)))
						}
						if r.App.Secrets[0].Name != "TEST" {
							return errors.New(fmt.Sprintf("Unexpected secret name %v", r.App.Secrets[0].Name))
						}
						return nil
					},
					resource.TestCheckResourceAttrWith(resourceName, "secrets.TEST.digest", func(value string) error {
						r, err := providerGraphql.GetSecrets(context.Background(), client, name)
						if err != nil {
							return err
						}
						if r.App.Secrets[0].Digest != value {
							return errors.New(fmt.Sprintf("digest in state (%s) differs from digest from fly API (%s)", value, r.App.Secrets[0].Digest))
						}
						if value != lastDigest {
							return errors.New(fmt.Sprintf("digest in state (%s) differs from digest of same value before drift (%s)", value, lastDigest))
						}
						return nil
					}),
				),
			},

			// Verify that we don't touch unmanaged secrets
			{
				PreConfig: func() {
					_, err := providerGraphql.SetSecrets(context.Background(), client, providerGraphql.SetSecretsInput{
						AppId: name,
						Secrets: []providerGraphql.SecretInput{{
							Key:   "UNMANAGED",
							Value: "1",
						}},
					})
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
	secrets = {
		TEST = {value = "3"}
	}
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "secrets.TEST.digest"),
					resource.TestCheckNoResourceAttr(resourceName, "secrets.UNMANAGED.digest"),
					func(state *terraform.State) error {
						r, err := providerGraphql.GetSecrets(context.Background(), client, name)
						if err != nil {
							return err
						}

						if len(r.App.Secrets) == 1 && r.App.Secrets[0].Name == "TEST" {
							return errors.New("unmanaged secret disappeared")
						} else if len(r.App.Secrets) != 2 {
							return errors.New(fmt.Sprintf("unexpected secrets in API %v", r.App.Secrets))
						}
						return nil
					},
				),
			},

			{
				Config: fmt.Sprintf(`
%s
resource "fly_app" "%s" {
	name = "%s"
	org = "%s"
	secrets = {}
}
`, providerConfig(), appName, name, getTestOrg()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "secrets.TEST"),
				),
			},
		},
	})
}
