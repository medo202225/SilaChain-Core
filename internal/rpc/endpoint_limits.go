package rpc

import "time"

var (
	txSendRateLimiter = NewRateLimiter(10*time.Second, 20)
	mineRateLimiter   = NewRateLimiter(30*time.Second, 3)
	faucetRateLimiter = NewRateLimiter(1*time.Minute, 5)
)
