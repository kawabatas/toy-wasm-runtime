package vm

import (
	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

type (
	Function interface {
		Call(vm *VM)
		// In the current version of WebAssembly,
		// the length of the result type vector of a valid function type may be at most 1.
		// This restriction may be removed in future versions.
		HasResult() bool
	}

	HostFunction struct {
		FdWrite *wasiFdWrite
	}

	WasmFunction struct {
		FunctionType            *wasm.FunctionType
		BodyOffsetInCodeSection uint64
		Body                    []byte
		Blocks                  map[uint64]*WasmFunctionBlock
	}

	WasmFunctionBlock struct {
		StartAt, ElseAt, EndAt uint64
		BlockType              *wasm.FunctionType
		BlockTypeBytes         uint64
	}
)

var (
	_ Function = (*HostFunction)(nil)
	_ Function = (*WasmFunction)(nil)
)

func (f *HostFunction) HasResult() bool {
	return false
}

func (f *HostFunction) Call(vm *VM) {
	// only support $fd_write (param i32 i32 i32 i32) (result i32)
	in := make([]int32, 4)
	for i := len(in) - 1; i >= 0; i-- {
		raw := vm.stack.Pop()
		in[i] = int32(raw)
	}

	f.FdWrite.Call(in[0], in[1], in[2], in[3])
}

func (f *WasmFunction) HasResult() bool {
	return len(f.FunctionType.Results) > 0
}

func (f *WasmFunction) Call(vm *VM) {
	paramCount := len(f.FunctionType.Params)
	locals := make([]uint64, f.BodyOffsetInCodeSection+uint64(paramCount))
	for i := 0; i < paramCount; i++ {
		locals[paramCount-1-i] = vm.stack.Pop()
	}

	prev := vm.activeFrame
	vm.activeFrame = NewFrame(f, locals)
	vm.invokeActiveFunction()
	vm.activeFrame = prev
}

func (vm *VM) invokeActiveFunction() {
	var total int
	for ; int(vm.activeFrame.PC) < len(vm.activeFrame.Function.Body); vm.activeFrame.PC++ {
		total++
		op := vm.activeFrame.Function.Body[vm.activeFrame.PC]
		switch wasm.Opcode(op) {
		case wasm.OpcodeReturn:
			return
		default:
			f, ok := instructionMap[op]
			if !ok {
				panic("vm instruction not defined")
			}
			f(vm)
		}

		// forced termination
		if total == 100000 {
			break
		}
	}
}
