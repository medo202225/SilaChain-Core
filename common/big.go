// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"math/big"

	"github.com/holiman/uint256"
)

// Shared numeric values used across SilaChain.
var (
	SilaBigZero          = big.NewInt(0)
	SilaBigOne           = big.NewInt(1)
	SilaBigTwo           = big.NewInt(2)
	SilaBigThree         = big.NewInt(3)
	SilaBigThirtyTwo     = big.NewInt(32)
	SilaBigTwoFiftySix   = big.NewInt(256)
	SilaBigTwoFiftySeven = big.NewInt(257)

	SilaUint256Zero = uint256.NewInt(0)
)
