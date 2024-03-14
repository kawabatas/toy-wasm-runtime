package vm

import (
	"bytes"
	"fmt"

	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

// Execution
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#execution%E2%91%A1
type (
	VM struct {
		Store *Store

		stack       *Stack
		activeFrame *Frame
	}

	Store struct {
		ModuleInstance *wasm.Module
		Functions      []Function
		Memory         []byte
	}
)

func InstantiateModule(module *wasm.Module) (*VM, error) {
	vm := &VM{
		Store: &Store{
			ModuleInstance: module,
		},
		stack: NewStack(),
	}

	if err := vm.initMemory(); err != nil {
		panic(err)
	}

	if err := vm.initFunctions(); err != nil {
		panic(err)
	}

	return vm, nil
}

func (vm *VM) initMemory() error {
	mem := make([]byte, wasm.MemoryPageSize)

	for _, ds := range vm.Store.ModuleInstance.DataSection {
		r := bytes.NewBuffer(ds.OffsetExpression.Data)
		switch ds.OffsetExpression.Opcode {
		case wasm.OpcodeI32Const:
			offset, _, err := wasm.DecodeInt32(r)
			if err != nil {
				return fmt.Errorf("decode int32 error: %w", err)
			}

			size := int(offset) + len(ds.Init)
			if uint32(size) > wasm.MemoryPageSize {
				return fmt.Errorf("memory size out of limit")
			}
			copy(mem[offset:], ds.Init)

		default:
			return fmt.Errorf("invalid opcode: %#x", ds.OffsetExpression.Opcode)
		}
	}

	vm.Store.Memory = mem
	return nil
}

func (vm *VM) initFunctions() error {
	m := vm.Store.ModuleInstance

	funcs := make([]Function, len(m.TypeSection))
	funcsIndex := 0

	for _, imp := range m.ImportSection {
		if imp.Module == wasiPreview1 && imp.Type == wasm.ExternTypeFunc {
			funcs[funcsIndex] = &HostFunction{
				FdWrite: newWasiFdWrite(vm),
			}
			funcsIndex++
		}
	}

	for i, fidx := range m.FunctionSection {
		f := &WasmFunction{
			FunctionType:            &m.TypeSection[fidx],
			Body:                    m.CodeSection[i].Body,
			BodyOffsetInCodeSection: m.CodeSection[i].BodyOffsetInCodeSection,
		}
		blocks, err := vm.parseBlocks(f.Body)
		if err != nil {
			return fmt.Errorf("parse blocks: %w", err)
		}
		f.Blocks = blocks
		funcs[funcsIndex] = f
		funcsIndex++
	}

	vm.Store.Functions = funcs
	return nil
}

func (vm *VM) InvokeFunction(name string, args ...uint64) (uint64, error) {
	funcs := vm.Store.Functions
	exp, ok := vm.Store.ModuleInstance.Exports[name]
	if !ok {
		return 0, fmt.Errorf("export func %s is not found", name)
	}

	if exp.Type != wasm.ExternTypeFunc {
		return 0, fmt.Errorf("export func %s is not func type", name)
	}

	if int(exp.Index) >= len(funcs) {
		return 0, fmt.Errorf("export func index out of range")
	}

	for _, arg := range args {
		vm.stack.Push(arg)
	}

	f := funcs[exp.Index]
	f.Call(vm)

	var ret uint64
	if f.HasResult() {
		ret = vm.stack.Pop()
	}
	return ret, nil
}

func (vm *VM) FetchInt32() int32 {
	r := bytes.NewBuffer(vm.activeFrame.Function.Body[vm.activeFrame.PC:])
	ret, num, err := wasm.DecodeInt32(r)
	if err != nil {
		panic(err)
	}
	vm.activeFrame.PC += num - 1 // 1-1=0
	return ret
}

func (vm *VM) FetchUint32() uint32 {
	r := bytes.NewBuffer(vm.activeFrame.Function.Body[vm.activeFrame.PC:])
	ret, num, err := wasm.DecodeUint32(r)
	if err != nil {
		panic(err)
	}
	vm.activeFrame.PC += num - 1 // 1-1=0
	return ret
}

type BlockType = wasm.FunctionType

func (vm *VM) parseBlocks(body []byte) (map[uint64]*WasmFunctionBlock, error) {
	ret := map[uint64]*WasmFunctionBlock{}
	stack := make([]*WasmFunctionBlock, 0)

	for pc := uint64(0); pc < uint64(len(body)); pc++ {
		rawOc := body[pc]

		switch wasm.Opcode(rawOc) {
		case wasm.OpcodeCall:
			pc++
			_, num, err := wasm.DecodeUint32(bytes.NewBuffer(body[pc:]))
			if err != nil {
				return nil, fmt.Errorf("read immediate: %w", err)
			}
			pc += num - 1
			continue
		case wasm.OpcodeI32Const:
			pc++
			_, num, err := wasm.DecodeInt32(bytes.NewBuffer(body[pc:]))
			if err != nil {
				return nil, fmt.Errorf("read immediate: %w", err)
			}
			pc += num - 1
			continue
		case wasm.OpcodeIf:
			var bt BlockType
			r := bytes.NewBuffer(body[pc+1:])
			raw, num, err := wasm.DecodeInt33AsInt64(r)
			if err != nil {
				return nil, fmt.Errorf("decode int33: %w", err)
			}
			switch raw {
			case -64: // 0x40 in original byte = nil
				bt = BlockType{}
			case -1: // 0x7f in original byte = i32
				bt = BlockType{Results: []wasm.ValueType{wasm.ValueTypeI32}}
			default:
				m := vm.Store.ModuleInstance
				if raw < 0 || (raw >= int64(len(m.TypeSection))) {
					return nil, fmt.Errorf("invalid block type: %d", raw)
				}
				bt = m.TypeSection[raw]
			}

			stack = append(stack, &WasmFunctionBlock{
				StartAt:        pc,
				BlockType:      &bt,
				BlockTypeBytes: num,
			})
			pc += num
		case wasm.OpcodeElse:
			stack[len(stack)-1].ElseAt = pc
		case wasm.OpcodeEnd:
			if int(pc) == len(body)-1 { // ignore last
				continue
			}
			bl := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			bl.EndAt = pc
			ret[bl.StartAt] = bl
		default:
			continue
		}
	}

	if len(stack) > 0 {
		return nil, fmt.Errorf("ill-nested block exists")
	}

	return ret, nil
}
