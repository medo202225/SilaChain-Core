package sdk

func NewDeployTx(req DeployTxRequest) DeployTxRequest {
	if req.VMVersion == 0 {
		req.VMVersion = 1
	}
	return req
}

func NewCallTx(req CallTxRequest) CallTxRequest {
	if req.VMVersion == 0 {
		req.VMVersion = 1
	}
	return req
}

func NewReadOnlyCall(req ReadOnlyCallRequest) JSONRPCRequest {
	if req.VMVersion == 0 {
		req.VMVersion = 1
	}
	if req.GasLimit == 0 {
		req.GasLimit = 100000
	}

	return JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "sila_call",
		Params: []any{
			map[string]any{
				"to":         req.To,
				"input":      req.Input,
				"vm_version": req.VMVersion,
				"gas_limit":  req.GasLimit,
			},
		},
	}
}
