package sdk

import (
	"net/http"
)

func (c *Client) SendTx(req any) (*http.Response, error) {
	return c.PostJSON("/tx/send", req)
}

func (c *Client) DeployContract(req DeployTxRequest) (*http.Response, error) {
	return c.PostJSON("/contract/deploy", req)
}

func (c *Client) CallContract(req CallTxRequest) (*http.Response, error) {
	return c.PostJSON("/contract/call", req)
}

func (c *Client) ReadOnlyCall(req ReadOnlyCallRequest) (*http.Response, error) {
	payload := NewReadOnlyCall(req)
	return c.PostJSON("/sila-rpc", payload)
}
