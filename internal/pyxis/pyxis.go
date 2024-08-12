package pyxis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/shurcooL/graphql"
)

const (
	DefaultPyxisHost = "catalog.redhat.com/api/containers"
)

type PyxisClient struct {
	Client    *http.Client
	PyxisHost string
}

func (p *PyxisClient) getPyxisGraphqlURL() string {
	return fmt.Sprintf("https://%s/graphql/", p.PyxisHost)
}

func NewPyxisClient(pyxisHost string, httpClient *http.Client) *PyxisClient {
	return &PyxisClient{
		Client:    httpClient,
		PyxisHost: pyxisHost,
	}
}

func (p *PyxisClient) FindOperatorIndices(ctx context.Context, organization string) ([]OperatorIndex, error) {
	// our graphQL query
	var query struct {
		FindOperatorIndices struct {
			OperatorIndex []struct {
				OCPVersion   graphql.String `graphql:"ocp_version"`
				Organization graphql.String `graphql:"organization"`
				EndOfLife    graphql.String `graphql:"end_of_life"`
			} `graphql:"data"`
			Errors struct {
				Status graphql.Int    `graphql:"status"`
				Detail graphql.String `graphql:"detail"`
			} `graphql:"error"`
			Total graphql.Int
			Page  graphql.Int
			// filter to make sure we get exact results, end_of_life is a string, querying for `null` yields active OCP versions.
		} `graphql:"find_operator_indices(filter:{and:[{organization:{eq:$organization}},{end_of_life:{eq:null}}]})"`
	}

	// variables to feed to our graphql filter
	variables := map[string]interface{}{
		"organization": graphql.String(organization),
	}

	// make our query
	client := graphql.NewClient(p.getPyxisGraphqlURL(), p.Client)

	err := client.Query(ctx, &query, variables)
	if err != nil {
		return nil, fmt.Errorf("error while executing remote query for %s catalogs: %v", organization, err)
	}

	operatorIndices := make([]OperatorIndex, len(query.FindOperatorIndices.OperatorIndex))
	for idx, operator := range query.FindOperatorIndices.OperatorIndex {
		operatorIndices[idx] = OperatorIndex{
			OCPVersion:   string(operator.OCPVersion),
			Organization: string(operator.Organization),
		}
	}

	return operatorIndices, nil
}
