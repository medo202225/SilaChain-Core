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

package state

import "github.com/SILA/sila-chain/metrics"

var (
accountReadMeters        = metrics.NewRegisteredMeter("sila/state/read/account", nil)
storageReadMeters        = metrics.NewRegisteredMeter("sila/state/read/storage", nil)
accountUpdatedMeter      = metrics.NewRegisteredMeter("sila/state/update/account", nil)
storageUpdatedMeter      = metrics.NewRegisteredMeter("sila/state/update/storage", nil)
accountDeletedMeter      = metrics.NewRegisteredMeter("sila/state/delete/account", nil)
storageDeletedMeter      = metrics.NewRegisteredMeter("sila/state/delete/storage", nil)
accountTrieUpdatedMeter  = metrics.NewRegisteredMeter("sila/state/update/accountnodes", nil)
storageTriesUpdatedMeter = metrics.NewRegisteredMeter("sila/state/update/storagenodes", nil)
accountTrieDeletedMeter  = metrics.NewRegisteredMeter("sila/state/delete/accountnodes", nil)
storageTriesDeletedMeter = metrics.NewRegisteredMeter("sila/state/delete/storagenodes", nil)
)
