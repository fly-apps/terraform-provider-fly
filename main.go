package main

import (
	"context"
	"flag"
	"log"

	"github.com/fly-apps/terraform-provider-fly/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"

    "github.com/pulumi/pulumi-terraform-bridge/pf/tfgen"

	pf "github.com/pulumi/pulumi-terraform-bridge/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
)

// Run "go generate" to format example terraform files and generate the docs for the registry/website

// If you do not have terraform installed, you can remove the formatting command, but its suggested to
// ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"

	// goreleaser can also pass the specific commit if you want
	// commit  string = ""
)

func Provider() pf.ProviderInfo {
	info := tfbridge.ProviderInfo{
		Name:    "fly",
		Version: version,
		Resources: map[string]*tfbridge.ResourceInfo{
			"myresource": {Tok: "myprovider::MyResource"},
		},
	}
	return pf.ProviderInfo{
		ProviderInfo: info,
		NewProvider: func() shim.Provider {
			return shim.New(version)
		},
	}
}

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.BoolVar(&pulumi, "pulumi", true, "set to true to pulumi gen")
    flag.Parse()

    if flag.pulumi {
        tfgen.Main("myprovider", Provider())
    } else {
        opts := providerserver.ServeOpts{
            Address: "registry.terraform.io/fly-apps/fly",
            Debug:   debug,
        }

        err := providerserver.Serve(context.Background(), provider.New(version), opts)

        if err != nil {
            log.Fatal(err.Error())
        }
    }
}
