package utils

import (
	"context"
	"net/http"
)

type Transport struct {
	UnderlyingTransport http.RoundTripper
	Token               string
	Ctx                 context.Context
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
	return t.UnderlyingTransport.RoundTrip(req)
}
