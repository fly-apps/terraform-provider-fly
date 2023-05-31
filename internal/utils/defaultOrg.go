package utils

import (
	"context"
	"errors"
	graphql2 "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
)

func GetDefaultOrg(ctx context.Context, client graphql2.Client) (*graphql.OrgsQueryOrganizationsOrganizationConnectionNodesOrganization, error) {
	queryresp, err := graphql.OrgsQuery(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(queryresp.Organizations.Nodes) > 1 {
		return nil, errors.New("organization is ambiguous. Your account has more than one organization, you must specify which to use")
	}
	if len(queryresp.Organizations.Nodes) < 1 {
		return nil, errors.New("no organizations to choose from. This error should not be reachable, if you find it please file an issue")
	}
	return &queryresp.Organizations.Nodes[0], nil
}
