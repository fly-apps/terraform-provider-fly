package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"os"
	"testing"
)

const providerConfigTemplate = `
provider "fly" {}
`

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"fly": providerserver.NewProtocol6WithError(New("test")()),
}

func providerConfig() string {
	return providerConfigTemplate
}

func regionConfig() string {
	region := os.Getenv("FLY_TF_TEST_REGION")
	if len(region) < 1 {
		region = "ewr"
	}
	return region
}

func testAccPreCheck(t *testing.T) {
	_, hasKey := os.LookupEnv("FLY_API_TOKEN")
	if !hasKey {
		t.Fatalf("Need api key in FLY_API_TOKEN")
	}
	_, hasApp := os.LookupEnv("FLY_TF_TEST_APP")
	if !hasApp {
		t.Fatalf("Need app in FLY_TF_TEST_APP")
	}
	_, hasOrg := os.LookupEnv("FLY_TF_TEST_ORG")
	if !hasOrg {
		t.Fatalf("Need org slug in FLY_TF_TEST_ORG")
	}
}
