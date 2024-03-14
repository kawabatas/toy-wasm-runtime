package vm

import (
	"encoding/binary"

	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

var instructionMap = map[wasm.Opcode]func(vm *VM){
	wasm.OpcodeUnreachable: func(vm *VM) { panic("unreachable") },
	wasm.OpcodeNop:         func(vm *VM) {},
	wasm.OpcodeIf:          ifInst,
	wasm.OpcodeElse:        elseInst,
	wasm.OpcodeEnd:         func(vm *VM) {},
	wasm.OpcodeReturn:      func(vm *VM) {},
	wasm.OpcodeCall:        call,
	wasm.OpcodeDrop:        drop,
	wasm.OpcodeLocalGet:    localGet,
	wasm.OpcodeI32Load:     i32Load,
	wasm.OpcodeI32Store:    i32Store,
	wasm.OpcodeI32Const:    i32Const,
	wasm.OpcodeI32Lts:      i32Lts,
	wasm.OpcodeI32Add:      i32Add,
	wasm.OpcodeI32Sub:      i32Sub,
}

func ifInst(vm *VM) {
	ctx := vm.activeFrame
	block, ok := ctx.Function.Blocks[vm.activeFrame.PC]
	if !ok {
		panic("function block not initialized")
	}
	ctx.PC += block.BlockTypeBytes

	if vm.stack.Pop() == 0 {
		// enter else
		vm.activeFrame.PC = block.ElseAt
	}

	ctx.LabelStack.Push(&Label{
		Arity:          len(block.BlockType.Results),
		ContinuationPC: block.EndAt,
		EndPC:          block.EndAt,
	})
}

func elseInst(vm *VM) {
	l := vm.activeFrame.LabelStack.Pop()
	vm.activeFrame.PC = l.EndPC
}

func call(vm *VM) {
	vm.activeFrame.PC++
	index := vm.FetchUint32()
	vm.Store.Functions[index].Call(vm)
}

func drop(vm *VM) {
	vm.stack.Drop()
}

func localGet(vm *VM) {
	vm.activeFrame.PC++
	id := vm.FetchUint32()
	vm.stack.Push(vm.activeFrame.Locals[id])
}

func _memoryBase(vm *VM) uint64 {
	vm.activeFrame.PC++
	_ = vm.FetchUint32() // ignore memory align
	vm.activeFrame.PC++
	return uint64(vm.FetchUint32()) + vm.stack.Pop()
}

func i32Load(vm *VM) {
	base := _memoryBase(vm)
	vm.stack.Push(uint64(binary.LittleEndian.Uint32(vm.Store.Memory[base:])))
}

func i32Store(vm *VM) {
	val := vm.stack.Pop()
	base := _memoryBase(vm)
	binary.LittleEndian.PutUint32(vm.Store.Memory[base:], uint32(val))
}

func i32Const(vm *VM) {
	vm.activeFrame.PC++
	vm.stack.Push(uint64(vm.FetchInt32()))
}

func i32Lts(vm *VM) {
	v2 := vm.stack.Pop()
	v1 := vm.stack.Pop()
	if v1 < v2 {
		vm.stack.Push(1)
	} else {
		vm.stack.Push(0)
	}
}

func i32Add(vm *VM) {
	v2 := vm.stack.Pop()
	v1 := vm.stack.Pop()
	vm.stack.Push(uint64(v1 + v2))
}

func i32Sub(vm *VM) {
	v2 := vm.stack.Pop()
	v1 := vm.stack.Pop()
	vm.stack.Push(uint64(v1 - v2))
}
