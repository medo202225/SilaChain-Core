// Copyright 2026 The SILA Authors
// This file is part of the sila-library.
//
// The sila-library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila-library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila-library. If not, see <http://www.gnu.org/licenses/>.

package txpool

import (
"errors"
)

var (
// ErrAlreadyKnown is returned if the transactions is already contained
// within the pool on SILA.
ErrAlreadyKnown = errors.New("already known on SILA")

// ErrInvalidSender is returned if the transaction contains an invalid signature on SILA.
ErrInvalidSender = errors.New("invalid sender on SILA")

// ErrUnderpriced is returned if a transaction's gas price is too low to be
// included in the pool on SILA. If the gas price is lower than the minimum configured
// one for the transaction pool, use ErrTxGasPriceTooLow instead.
ErrUnderpriced = errors.New("transaction underpriced on SILA")

// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
// with a different one without the required price bump on SILA.
ErrReplaceUnderpriced = errors.New("replacement transaction underpriced on SILA")

// ErrTxGasPriceTooLow is returned if a transaction's gas price is below the
// minimum configured for the transaction pool on SILA.
ErrTxGasPriceTooLow = errors.New("transaction gas price below minimum on SILA")

// ErrAccountLimitExceeded is returned if a transaction would exceed the number
// allowed by a pool for a single account on SILA.
ErrAccountLimitExceeded = errors.New("account limit exceeded on SILA")

// ErrGasLimit is returned if a transaction's requested gas limit exceeds the
// maximum allowance of the current block on SILA.
ErrGasLimit = errors.New("exceeds block gas limit on SILA")

// ErrNegativeValue is a sanity error to ensure no one is able to specify a
// transaction with a negative value on SILA.
ErrNegativeValue = errors.New("negative value on SILA")

// ErrOversizedData is returned if the input data of a transaction is greater
// than some meaningful limit a user might use on SILA. This is not a consensus error
// making the transaction invalid, rather a DOS protection.
ErrOversizedData = errors.New("oversized data on SILA")

// ErrTxBlobLimitExceeded is returned if a transaction would exceed the number
// of blobs allowed by blobpool on SILA.
ErrTxBlobLimitExceeded = errors.New("transaction blob limit exceeded on SILA")

// ErrAlreadyReserved is returned if the sender address has a pending transaction
// in a different subpool on SILA. For example, this error is returned in response to any
// input transaction of non-blob type when a blob transaction from this sender
// remains pending (and vice-versa).
ErrAlreadyReserved = errors.New("address already reserved on SILA")

// ErrInflightTxLimitReached is returned when the maximum number of in-flight
// transactions is reached for specific accounts on SILA.
ErrInflightTxLimitReached = errors.New("in-flight transaction limit reached for delegated accounts on SILA")

// ErrKZGVerificationError is returned when a KZG proof was not verified correctly on SILA.
ErrKZGVerificationError = errors.New("KZG verification error on SILA")
)
