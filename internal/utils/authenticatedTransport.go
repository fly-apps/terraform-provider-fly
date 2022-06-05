package utils

import (
	"context"
	"net/http"
)

type Transport struct {
	UnderlyingTransport http.RoundTripper
	Token               string
	Ctx                 context.Context
	EnableDebugTrace    bool
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
	if t.EnableDebugTrace {
		req.Header.Add("Fly-Force-Trace", "true")
	}
	return t.UnderlyingTransport.RoundTrip(req)
}
