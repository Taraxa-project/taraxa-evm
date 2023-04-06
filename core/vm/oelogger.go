package vm

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
)

const (
	CALL_string         = "call"
	CALLCODE_string     = "callcode"
	DELEGATECALL_string = "delegatecall"
	STATICCALL_string   = "staticcall"
	CREATE_string       = "create"
	SUICIDE             = "suicide"
)

type TracingConfig struct {
	VmTrace   bool
	Trace     bool
	StateDiff bool
}

type ParityTrace struct {
	// Do not change the ordering of these fields -- allows for easier comparison with other clients
	Action              interface{}  `json:"action"` // Can be either CallTraceAction or CreateTraceAction
	BlockHash           *common.Hash `json:"blockHash,omitempty"`
	BlockNumber         *uint64      `json:"blockNumber,omitempty"`
	Error               string       `json:"error,omitempty"`
	Result              interface{}  `json:"result"`
	Subtraces           int          `json:"subtraces"`
	TraceAddress        []int        `json:"traceAddress"`
	TransactionHash     *common.Hash `json:"transactionHash,omitempty"`
	TransactionPosition *uint64      `json:"transactionPosition,omitempty"`
	Type                string       `json:"type"`
}

// TraceCallResult is the response to `trace_call` method
type TraceCallResult struct {
	Output          hexutil.Bytes                        `json:"output"`
	StateDiff       map[common.Address]*StateDiffAccount `json:"stateDiff"`
	Trace           []*ParityTrace                       `json:"trace"`
	VmTrace         *VmTrace                             `json:"vmTrace"`
	TransactionHash *common.Hash                         `json:"transactionHash,omitempty"`
}

// StateDiffAccount is the part of `trace_call` response that is under "stateDiff" tag
type StateDiffAccount struct {
	Balance interface{}                            `json:"balance"` // Can be either string "=" or mapping "*" => {"from": "hex", "to": "hex"}
	Code    interface{}                            `json:"code"`
	Nonce   interface{}                            `json:"nonce"`
	Storage map[common.Hash]map[string]interface{} `json:"storage"`
}

type StateDiffBalance struct {
	From *hexutil.Big `json:"from"`
	To   *hexutil.Big `json:"to"`
}

type StateDiffCode struct {
	From hexutil.Bytes `json:"from"`
	To   hexutil.Bytes `json:"to"`
}

type StateDiffNonce struct {
	From hexutil.Uint64 `json:"from"`
	To   hexutil.Uint64 `json:"to"`
}

type StateDiffStorage struct {
	From common.Hash `json:"from"`
	To   common.Hash `json:"to"`
}

// VmTrace is the part of `trace_call` response that is under "vmTrace" tag
type VmTrace struct {
	Code hexutil.Bytes `json:"code"`
	Ops  []*VmTraceOp  `json:"ops"`
}

// VmTraceOp is one element of the vmTrace ops trace
type VmTraceOp struct {
	Cost int        `json:"cost"`
	Ex   *VmTraceEx `json:"ex"`
	Pc   int        `json:"pc"`
	Sub  *VmTrace   `json:"sub"`
	Op   string     `json:"op,omitempty"`
	Idx  string     `json:"idx,omitempty"`
}

type VmTraceEx struct {
	Mem   *VmTraceMem   `json:"mem"`
	Push  []string      `json:"push"`
	Store *VmTraceStore `json:"store"`
	Used  int           `json:"used"`
}

type VmTraceMem struct {
	Data string `json:"data"`
	Off  int    `json:"off"`
}

type VmTraceStore struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// TraceAction A parity formatted trace action
type TraceAction struct {
	// Do not change the ordering of these fields -- allows for easier comparison with other clients
	Author         string         `json:"author,omitempty"`
	RewardType     string         `json:"rewardType,omitempty"`
	SelfDestructed string         `json:"address,omitempty"`
	Balance        string         `json:"balance,omitempty"`
	CallType       string         `json:"callType,omitempty"`
	From           common.Address `json:"from"`
	Gas            hexutil.Big    `json:"gas"`
	Init           hexutil.Bytes  `json:"init,omitempty"`
	Input          hexutil.Bytes  `json:"input,omitempty"`
	RefundAddress  string         `json:"refundAddress,omitempty"`
	To             string         `json:"to,omitempty"`
	Value          string         `json:"value,omitempty"`
}

type CallTraceAction struct {
	From     common.Address `json:"from"`
	CallType string         `json:"callType"`
	Gas      hexutil.Big    `json:"gas"`
	Input    hexutil.Bytes  `json:"input"`
	To       common.Address `json:"to"`
	Value    hexutil.Big    `json:"value"`
}

type CreateTraceAction struct {
	From  common.Address `json:"from"`
	Gas   hexutil.Big    `json:"gas"`
	Init  hexutil.Bytes  `json:"init"`
	Value hexutil.Big    `json:"value"`
}

type SuicideTraceAction struct {
	Address       common.Address `json:"address"`
	RefundAddress common.Address `json:"refundAddress"`
	Balance       hexutil.Big    `json:"balance"`
}

type CreateTraceResult struct {
	// Do not change the ordering of these fields -- allows for easier comparison with other clients
	Address *common.Address `json:"address,omitempty"`
	Code    hexutil.Bytes   `json:"code"`
	GasUsed *hexutil.Big    `json:"gasUsed"`
}

// TraceResult A parity formatted trace result
type TraceResult struct {
	// Do not change the ordering of these fields -- allows for easier comparison with other clients
	GasUsed *hexutil.Big  `json:"gasUsed"`
	Output  hexutil.Bytes `json:"output"`
}

// OpenEthereum-style tracer
type OeTracer struct {
	r            *TraceCallResult
	traceAddr    []int
	traceStack   []*ParityTrace
	precompile   bool // Whether the last CaptureStart was called with `precompile = true`
	compat       bool // Bug for bug compatibility mode
	lastVmOp     *VmTraceOp
	lastOp       OpCode
	lastMemOff   uint64
	lastMemLen   uint64
	memOffStack  []uint64
	memLenStack  []uint64
	lastOffStack *VmTraceOp
	vmOpStack    []*VmTraceOp // Stack of vmTrace operations as call depth increases
	idx          []string     // Prefix for the "idx" inside operations, for easier navigation
}

// NewStructLogger returns a new eologger
func NewOeTracer(cfg *TracingConfig) *OeTracer {
	logger := &OeTracer{}
	if cfg != nil {
		traceResult := &TraceCallResult{Trace: []*ParityTrace{}}
		if cfg.VmTrace {
			traceResult.VmTrace = &VmTrace{Ops: []*VmTraceOp{}}
		}
		if cfg.Trace || cfg.VmTrace {
			logger.r = traceResult
			logger.traceAddr = []int{}
		}
	}
	return logger
}

func (ot *OeTracer) SetRetCode(output []byte) {
	ot.r.Output = common.CopyBytes(output)
}

func (ot *OeTracer) GetResult() *TraceCallResult {
	return ot.r
}

func (ot *OeTracer) captureStartOrEnter(deep bool, typ OpCode, from *common.Address, to *common.Address, precompile bool, create bool, input []byte, gas uint64, value *big.Int, code []byte) {
	//fmt.Printf("captureStartOrEnter deep %t, typ %s, from %x, to %x, create %t, input %x, gas %d, value %d, precompile %t\n", deep, typ.String(), from, to, create, input, gas, value, precompile)
	if ot.r.VmTrace != nil {
		var vmTrace *VmTrace
		if deep {
			var vmT *VmTrace
			if len(ot.vmOpStack) > 0 {
				vmT = ot.vmOpStack[len(ot.vmOpStack)-1].Sub
			} else {
				vmT = ot.r.VmTrace
			}
			if !ot.compat {
				ot.idx = append(ot.idx, fmt.Sprintf("%d-", len(vmT.Ops)-1))
			}
		}
		if ot.lastVmOp != nil {
			vmTrace = &VmTrace{Ops: []*VmTraceOp{}}
			ot.lastVmOp.Sub = vmTrace
			ot.vmOpStack = append(ot.vmOpStack, ot.lastVmOp)
		} else {
			vmTrace = ot.r.VmTrace
		}
		if create {
			vmTrace.Code = common.CopyBytes(input)
			if ot.lastVmOp != nil {
				ot.lastVmOp.Cost += int(gas)
			}
		} else {
			vmTrace.Code = code
		}
	}
	if precompile && deep && (value == nil || len(value.Bits()) == 0) {
		ot.precompile = true
		return
	}
	if gas > 500000000 {
		gas = 500000001 - (0x8000000000000000 - gas)
	}
	trace := &ParityTrace{}
	if create {
		trResult := &CreateTraceResult{}
		trace.Type = CREATE_string
		trResult.Address = new(common.Address)
		copy(trResult.Address[:], to.Bytes())
		trace.Result = trResult
	} else {
		trace.Result = &TraceResult{}
		trace.Type = CALL_string
	}
	if deep {
		topTrace := ot.traceStack[len(ot.traceStack)-1]
		traceIdx := topTrace.Subtraces
		ot.traceAddr = append(ot.traceAddr, traceIdx)
		topTrace.Subtraces++
		if typ == DELEGATECALL {
			switch action := topTrace.Action.(type) {
			case *CreateTraceAction:
				value = action.Value.ToInt()
			case *CallTraceAction:
				value = action.Value.ToInt()
			}
		}
		if typ == STATICCALL {
			value = big.NewInt(0)
		}
	}
	trace.TraceAddress = make([]int, len(ot.traceAddr))
	copy(trace.TraceAddress, ot.traceAddr)
	if create {
		action := CreateTraceAction{}
		action.From = *from
		action.Gas.ToInt().SetUint64(gas)
		action.Init = common.CopyBytes(input)
		action.Value.ToInt().Set(value)
		trace.Action = &action
	} else if typ == SELFDESTRUCT {
		trace.Type = SUICIDE
		trace.Result = nil
		action := &SuicideTraceAction{}
		action.Address = *from
		action.RefundAddress = *to
		action.Balance.ToInt().Set(value)
		trace.Action = action
	} else {
		action := CallTraceAction{}
		switch typ {
		case CALL:
			action.CallType = CALL_string
		case CALLCODE:
			action.CallType = CALLCODE_string
		case DELEGATECALL:
			action.CallType = DELEGATECALL_string
		case STATICCALL:
			action.CallType = STATICCALL_string
		}
		action.From = *from
		action.To = *to
		action.Gas.ToInt().SetUint64(gas)
		action.Input = common.CopyBytes(input)
		action.Value.ToInt().Set(value)
		trace.Action = &action
	}
	ot.r.Trace = append(ot.r.Trace, trace)
	ot.traceStack = append(ot.traceStack, trace)
}

func (ot *OeTracer) CaptureStart(env *EVM, from *common.Address, to *common.Address, precompile bool, create bool, input []byte, gas uint64, value *big.Int, code []byte) error {
	ot.captureStartOrEnter(false /* deep */, CALL, from, to, precompile, create, input, gas, value, code)
	return nil
}

func (ot *OeTracer) CaptureEnter(typ OpCode, from *common.Address, to *common.Address, precompile bool, create bool, input []byte, gas uint64, value *big.Int, code []byte) error {
	ot.captureStartOrEnter(true /* deep */, typ, from, to, precompile, create, input, gas, value, code)
	return nil
}

func (ot *OeTracer) captureEndOrExit(deep bool, output []byte, usedGas uint64, err error) {
	if ot.r.VmTrace != nil {
		if len(ot.vmOpStack) > 0 {
			ot.lastOffStack = ot.vmOpStack[len(ot.vmOpStack)-1]
			ot.vmOpStack = ot.vmOpStack[:len(ot.vmOpStack)-1]
		}
		if !ot.compat && deep {
			ot.idx = ot.idx[:len(ot.idx)-1]
		}
		if deep {
			ot.lastMemOff = ot.memOffStack[len(ot.memOffStack)-1]
			ot.memOffStack = ot.memOffStack[:len(ot.memOffStack)-1]
			ot.lastMemLen = ot.memLenStack[len(ot.memLenStack)-1]
			ot.memLenStack = ot.memLenStack[:len(ot.memLenStack)-1]
		}
	}
	if ot.precompile {
		ot.precompile = false
		return
	}
	if !deep {
		ot.r.Output = common.CopyBytes(output)
	}
	ignoreError := false
	topTrace := ot.traceStack[len(ot.traceStack)-1]
	if ot.compat {
		ignoreError = !deep && topTrace.Type == CREATE_string
	}
	if err != nil && !ignoreError {
		if err == ErrExecutionReverted {
			topTrace.Error = "Reverted"
			switch topTrace.Type {
			case CALL_string:
				topTrace.Result.(*TraceResult).GasUsed = new(hexutil.Big)
				topTrace.Result.(*TraceResult).GasUsed.ToInt().SetUint64(usedGas)
				topTrace.Result.(*TraceResult).Output = common.CopyBytes(output)
			case CREATE_string:
				topTrace.Result.(*CreateTraceResult).GasUsed = new(hexutil.Big)
				topTrace.Result.(*CreateTraceResult).GasUsed.ToInt().SetUint64(usedGas)
				topTrace.Result.(*CreateTraceResult).Code = common.CopyBytes(output)
			}
		} else {
			topTrace.Result = nil
			switch err {
			case ErrContractAddressCollision, ErrCodeStoreOutOfGas, ErrOutOfGas, errGasUintOverflow:
				topTrace.Error = "Out of gas"
			case ErrWriteProtection:
				topTrace.Error = "Mutable Call In Static Context"
			default:
				topTrace.Error = err.Error()
			}
		}
	} else {
		if len(output) > 0 {
			switch topTrace.Type {
			case CALL_string:
				topTrace.Result.(*TraceResult).Output = common.CopyBytes(output)
			case CREATE_string:
				topTrace.Result.(*CreateTraceResult).Code = common.CopyBytes(output)
			}
		}
		switch topTrace.Type {
		case CALL_string:
			topTrace.Result.(*TraceResult).GasUsed = new(hexutil.Big)
			topTrace.Result.(*TraceResult).GasUsed.ToInt().SetUint64(usedGas)
		case CREATE_string:
			topTrace.Result.(*CreateTraceResult).GasUsed = new(hexutil.Big)
			topTrace.Result.(*CreateTraceResult).GasUsed.ToInt().SetUint64(usedGas)
		}
	}
	ot.traceStack = ot.traceStack[:len(ot.traceStack)-1]
	if deep {
		ot.traceAddr = ot.traceAddr[:len(ot.traceAddr)-1]
	}
}

func (ot *OeTracer) CaptureEnd(output []byte, usedGas uint64, t time.Duration, err error) error {
	ot.captureEndOrExit(false /* deep */, output, usedGas, err)
	return nil
}

func (ot *OeTracer) CaptureExit(output []byte, usedGas uint64, t time.Duration, err error) error {
	ot.captureEndOrExit(true /* deep */, output, usedGas, err)
	return nil
}

func (ot *OeTracer) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, st *Stack, contract *Contract, depth uint16, err error) error {
	if ot.r.VmTrace != nil {
		var vmTrace *VmTrace
		if len(ot.vmOpStack) > 0 {
			vmTrace = ot.vmOpStack[len(ot.vmOpStack)-1].Sub
		} else {
			vmTrace = ot.r.VmTrace
		}
		if ot.lastVmOp != nil && ot.lastVmOp.Ex != nil {
			// Set the "push" of the last operation
			var showStack int
			switch {
			case ot.lastOp >= PUSH1 && ot.lastOp <= PUSH32:
				showStack = 1
			case ot.lastOp >= SWAP1 && ot.lastOp <= SWAP16:
				showStack = int(ot.lastOp-SWAP1) + 2
			case ot.lastOp >= DUP1 && ot.lastOp <= DUP16:
				showStack = int(ot.lastOp-DUP1) + 2
			}
			switch ot.lastOp {
			case CALLDATALOAD, SLOAD, MLOAD, CALLDATASIZE, LT, GT, DIV, SDIV, SAR, AND, EQ, CALLVALUE, ISZERO,
				ADD, EXP, CALLER, KECCAK256, SUB, ADDRESS, GAS, MUL, RETURNDATASIZE, NOT, SHR, SHL,
				EXTCODESIZE, SLT, OR, NUMBER, PC, TIMESTAMP, BALANCE, SELFBALANCE, MULMOD, ADDMOD,
				BLOCKHASH, BYTE, XOR, ORIGIN, CODESIZE, MOD, SIGNEXTEND, GASLIMIT, DIFFICULTY, SGT, GASPRICE,
				MSIZE, EXTCODEHASH, SMOD, CHAINID, COINBASE:
				showStack = 1
			}
			for i := showStack - 1; i >= 0; i-- {
				if st.len() > i {
					ot.lastVmOp.Ex.Push = append(ot.lastVmOp.Ex.Push, st.Back(i).String())
				}
			}
			// Set the "mem" of the last operation
			var setMem bool
			switch ot.lastOp {
			case MSTORE, MSTORE8, MLOAD, RETURNDATACOPY, CALLDATACOPY, CODECOPY:
				setMem = true
			}
			if setMem && ot.lastMemLen > 0 {
				cpy := memory.GetCopy(int64(ot.lastMemOff), int64(ot.lastMemLen))
				if len(cpy) == 0 {
					cpy = make([]byte, ot.lastMemLen)
				}
				ot.lastVmOp.Ex.Mem = &VmTraceMem{Data: fmt.Sprintf("0x%0x", cpy), Off: int(ot.lastMemOff)}
			}
		}
		if ot.lastOffStack != nil {
			ot.lastOffStack.Ex.Used = int(gas)
			if st.len() > 0 {
				ot.lastOffStack.Ex.Push = []string{st.Back(0).String()}
			} else {
				ot.lastOffStack.Ex.Push = []string{}
			}
			if ot.lastMemLen > 0 && memory != nil {
				cpy := memory.GetCopy(int64(ot.lastMemOff), int64(ot.lastMemLen))
				if len(cpy) == 0 {
					cpy = make([]byte, ot.lastMemLen)
				}
				ot.lastOffStack.Ex.Mem = &VmTraceMem{Data: fmt.Sprintf("0x%0x", cpy), Off: int(ot.lastMemOff)}
			}
			ot.lastOffStack = nil
		}
		if ot.lastOp == STOP && op == STOP && len(ot.vmOpStack) == 0 {
			// Looks like OE is "optimising away" the second STOP
			return nil
		}
		ot.lastVmOp = &VmTraceOp{Ex: &VmTraceEx{}}
		vmTrace.Ops = append(vmTrace.Ops, ot.lastVmOp)
		if !ot.compat {
			var sb strings.Builder
			sb.Grow(len(ot.idx))
			for _, idx := range ot.idx {
				sb.WriteString(idx)
			}
			ot.lastVmOp.Idx = fmt.Sprintf("%s%d", sb.String(), len(vmTrace.Ops)-1)
		}
		ot.lastOp = op
		ot.lastVmOp.Cost = int(cost)
		ot.lastVmOp.Pc = int(pc)
		ot.lastVmOp.Ex.Push = []string{}
		ot.lastVmOp.Ex.Used = int(gas) - int(cost)
		if !ot.compat {
			ot.lastVmOp.Op = op.String()
		}
		switch op {
		case MSTORE, MLOAD:
			if st.len() > 0 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = 32
			}
		case MSTORE8:
			if st.len() > 0 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = 1
			}
		case RETURNDATACOPY, CALLDATACOPY, CODECOPY:
			if st.len() > 2 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = st.Back(2).Uint64()
			}
		case STATICCALL, DELEGATECALL:
			if st.len() > 5 {
				ot.memOffStack = append(ot.memOffStack, st.Back(4).Uint64())
				ot.memLenStack = append(ot.memLenStack, st.Back(5).Uint64())
			}
		case CALL, CALLCODE:
			if st.len() > 6 {
				ot.memOffStack = append(ot.memOffStack, st.Back(5).Uint64())
				ot.memLenStack = append(ot.memLenStack, st.Back(6).Uint64())
			}
		case CREATE, CREATE2, SELFDESTRUCT:
			// Effectively disable memory output
			ot.memOffStack = append(ot.memOffStack, 0)
			ot.memLenStack = append(ot.memLenStack, 0)
		case SSTORE:
			if st.len() > 1 {
				ot.lastVmOp.Ex.Store = &VmTraceStore{Key: st.Back(0).String(), Val: st.Back(1).String()}
			}
		}
		if ot.lastVmOp.Ex.Used < 0 {
			ot.lastVmOp.Ex = nil
		}
	}
	return nil
}
