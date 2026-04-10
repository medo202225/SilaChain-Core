package state

import (
	"strings"

	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

const (
	ContractCallBaseGas      coretypes.Gas = 30000
	ContractReadGas          coretypes.Gas = 5000
	ContractWriteGas         coretypes.Gas = 15000
	ContractDeleteGas        coretypes.Gas = 12000
	ContractCounterUpdateGas coretypes.Gas = 10000
)

type VMExecutionInput struct {
	VMVersion    uint16
	GasRemaining uint64
	ContractAddr string
	StorageAddr  string
	CodeAddr     string
	Caller       string
	Origin       string
	CallValue    uint64
	Input        []byte
	Code         []byte
}

type VMExecutionOutput struct {
	Success        bool
	GasUsed        uint64
	ReturnData     []byte
	RevertData     []byte
	CreatedAddress string
	Logs           []pkgtypes.Event
	Err            error
}

type VMExecutor interface {
	ExecuteContract(input VMExecutionInput) VMExecutionOutput
	GetContractCode(address string) []byte
}

type Transition struct {
	manager   *Manager
	contracts *ContractRegistry

	codeRegistry *ContractCodeRegistry
	storage      *ContractStorage
	journal      *Journal

	vmExecutor VMExecutor
}

func NewTransition(manager *Manager, contracts *ContractRegistry) *Transition {
	if contracts == nil {
		contracts = NewContractRegistry(manager, NewContractStorage())
	}

	codeRegistry := NewContractCodeRegistry()
	storage := NewContractStorage()
	journal := NewJournal()

	return &Transition{
		manager:      manager,
		contracts:    contracts,
		codeRegistry: codeRegistry,
		storage:      storage,
		journal:      journal,
	}
}

func (t *Transition) SetVMExecutor(executor VMExecutor) {
	if t == nil {
		return
	}
	t.vmExecutor = executor
}

func gasRequiredForTransaction(transaction *coretypes.Transaction) coretypes.Gas {
	if transaction == nil {
		return TransferIntrinsicGas
	}

	if transaction.IsContractDeploy() || transaction.UsesVM() {
		if transaction.GasLimit > 0 {
			return transaction.GasLimit
		}
		return ContractCallBaseGas + ContractWriteGas
	}

	if !transaction.IsContractCall() {
		return TransferIntrinsicGas
	}

	switch strings.ToLower(strings.TrimSpace(transaction.CallMethod)) {
	case "get":
		return ContractCallBaseGas + ContractReadGas
	case "set":
		return ContractCallBaseGas + ContractWriteGas
	case "delete":
		return ContractCallBaseGas + ContractDeleteGas
	case "inc", "dec":
		return ContractCallBaseGas + ContractCounterUpdateGas
	default:
		return ContractCallBaseGas + ContractWriteGas
	}
}

func (t *Transition) ApplyTransaction(transaction *coretypes.Transaction) error {
	_, err := t.ApplyTransactionWithResult(transaction)
	return err
}

func (t *Transition) ApplyTransactionWithResult(transaction *coretypes.Transaction) (Result, error) {
	if transaction == nil {
		return FailedResult("", "", "", 0, 0, 0, 0, 0, nil, "", "", "", 0, coretypes.ErrNilTransaction), coretypes.ErrNilTransaction
	}

	fromAcc, err := t.manager.GetAccount(transaction.From)
	if err != nil {
		return FailedResult(
			transaction.Hash,
			transaction.From,
			transaction.To,
			transaction.Value,
			transaction.EffectiveFee(),
			transaction.TotalCost(),
			0,
			0,
			nil,
			"",
			"",
			"",
			transaction.Timestamp,
			err,
		), err
	}

	var recipientLoaded bool
	if transaction.To != "" {
		_, err := t.manager.GetAccount(transaction.To)
		if err != nil && !transaction.IsContractDeploy() {
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				0,
				fromAcc.Nonce,
				nil,
				"",
				"",
				"",
				transaction.Timestamp,
				err,
			), err
		}
		if err == nil {
			recipientLoaded = true
		}
	}

	requiredGas := gasRequiredForTransaction(transaction)

	effectiveGasLimit := transaction.GasLimit
	if effectiveGasLimit == 0 {
		effectiveGasLimit = requiredGas
	}
	if effectiveGasLimit < requiredGas {
		return FailedResult(
			transaction.Hash,
			transaction.From,
			transaction.To,
			transaction.Value,
			transaction.EffectiveFee(),
			transaction.TotalCost(),
			effectiveGasLimit,
			fromAcc.Nonce,
			nil,
			"",
			"",
			"",
			transaction.Timestamp,
			coretypes.ErrInvalidGasLimit,
		), coretypes.ErrInvalidGasLimit
	}

	if err := coretypes.ValidateNonce(fromAcc.Nonce, transaction.Nonce); err != nil {
		return FailedResult(
			transaction.Hash,
			transaction.From,
			transaction.To,
			transaction.Value,
			transaction.EffectiveFee(),
			transaction.TotalCost(),
			requiredGas,
			fromAcc.Nonce,
			nil,
			"",
			"",
			"",
			transaction.Timestamp,
			err,
		), err
	}

	if err := fromAcc.Debit(transaction.TotalCost()); err != nil {
		return FailedResult(
			transaction.Hash,
			transaction.From,
			transaction.To,
			transaction.Value,
			transaction.EffectiveFee(),
			transaction.TotalCost(),
			requiredGas,
			fromAcc.Nonce,
			nil,
			"",
			"",
			"",
			transaction.Timestamp,
			err,
		), err
	}

	var logs []pkgtypes.Event
	gasUsed := requiredGas
	returnData := ""
	revertData := ""
	createdAddress := coretypes.Address("")

	if transaction.IsContractDeploy() || transaction.UsesVM() {
		if t.vmExecutor == nil {
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				requiredGas,
				fromAcc.Nonce,
				nil,
				"",
				"",
				"",
				transaction.Timestamp,
				coretypes.ErrUnsupportedTransactionType,
			), coretypes.ErrUnsupportedTransactionType
		}

		var codeBytes []byte
		if transaction.IsContractDeploy() {
			codeBytes = []byte(transaction.ContractCode)
		} else {
			codeBytes = t.vmExecutor.GetContractCode(string(transaction.To))
			if len(codeBytes) == 0 {
				return FailedResult(
					transaction.Hash,
					transaction.From,
					transaction.To,
					transaction.Value,
					transaction.EffectiveFee(),
					transaction.TotalCost(),
					requiredGas,
					fromAcc.Nonce,
					nil,
					"",
					"",
					"",
					transaction.Timestamp,
					coretypes.ErrUnsupportedTransactionType,
				), coretypes.ErrUnsupportedTransactionType
			}
		}

		output := t.vmExecutor.ExecuteContract(VMExecutionInput{
			VMVersion:    chooseVMVersion(transaction.VMVersion),
			GasRemaining: uint64(transaction.GasLimit),
			ContractAddr: string(transaction.To),
			StorageAddr:  string(transaction.To),
			CodeAddr:     string(transaction.To),
			Caller:       string(transaction.From),
			Origin:       string(transaction.From),
			CallValue:    uint64(transaction.Value),
			Input:        []byte(transaction.ContractInput),
			Code:         codeBytes,
		})

		if output.GasUsed > 0 {
			gasUsed = coretypes.Gas(output.GasUsed)
		}
		returnData = string(output.ReturnData)
		revertData = string(output.RevertData)
		createdAddress = coretypes.Address(output.CreatedAddress)
		logs = append(logs, output.Logs...)

		if !output.Success {
			vmErr := output.Err
			if vmErr == nil {
				vmErr = coretypes.ErrUnsupportedTransactionType
			}
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				gasUsed,
				fromAcc.Nonce,
				logs,
				returnData,
				revertData,
				createdAddress,
				transaction.Timestamp,
				vmErr,
			), vmErr
		}
	} else if transaction.IsContractCall() {
		callResult, err := t.contracts.Call(
			transaction.To,
			transaction.CallMethod,
			transaction.CallKey,
			transaction.CallValue,
		)
		if err != nil {
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				requiredGas,
				fromAcc.Nonce,
				nil,
				"",
				"",
				"",
				transaction.Timestamp,
				err,
			), err
		}
		logs = callResult.Logs
	} else {
		if !recipientLoaded {
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				requiredGas,
				fromAcc.Nonce,
				nil,
				"",
				"",
				"",
				transaction.Timestamp,
				coretypes.ErrUnsupportedTransactionType,
			), coretypes.ErrUnsupportedTransactionType
		}

		toAcc, err := t.manager.GetAccount(transaction.To)
		if err != nil {
			return FailedResult(
				transaction.Hash,
				transaction.From,
				transaction.To,
				transaction.Value,
				transaction.EffectiveFee(),
				transaction.TotalCost(),
				requiredGas,
				fromAcc.Nonce,
				nil,
				"",
				"",
				"",
				transaction.Timestamp,
				err,
			), err
		}
		toAcc.Credit(transaction.Value)
	}

	fromAcc.IncrementNonce()

	return SuccessResult(
		transaction.Hash,
		transaction.From,
		transaction.To,
		transaction.Value,
		transaction.EffectiveFee(),
		transaction.TotalCost(),
		gasUsed,
		fromAcc.Nonce,
		logs,
		returnData,
		revertData,
		createdAddress,
		transaction.Timestamp,
	), nil
}

func chooseVMVersion(v uint16) uint16 {
	if v == 0 {
		return 1
	}
	return v
}

func (t *Transition) CodeRegistry() *ContractCodeRegistry {
	if t == nil {
		return nil
	}
	return t.codeRegistry
}

func (t *Transition) Storage() *ContractStorage {
	if t == nil {
		return nil
	}
	return t.storage
}

func (t *Transition) Journal() *Journal {
	if t == nil {
		return nil
	}
	return t.journal
}
