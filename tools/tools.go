package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/wg"

	// Ensure documentation generator is not removed from go.mod.
	_ "github.com/Khan/genqlient/generate"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

type transport struct {
	underlyingTransport http.RoundTripper
	token               string
	ctx                 context.Context
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.token)
	return t.underlyingTransport.RoundTrip(req)
}

func main() {
	ctx := context.Background()
	token := os.Getenv("FLY_API_TOKEN")
	h := http.Client{Timeout: 60 * time.Second, Transport: &transport{underlyingTransport: http.DefaultTransport, token: token, ctx: ctx}}
	client := graphql.NewClient("https://api.fly.io/graphql", &h)
	tunnel, err := wg.Establish(ctx, "P7lZB0nw2ylg8smzmMLA9eVLAQuRL6", "ewr", token, &client)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	tunnelClient := tunnel.NewHttpClient()

	answer, err := tunnel.LookupAAAA(ctx, "google.com")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(answer)

	tresp, err := tunnelClient.HttpClient.Get(fmt.Sprintf("http://%s:4280/v1/apps/dovdotdev", "_api.internal"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(tresp.Status)
	tunnel.Down()
}
