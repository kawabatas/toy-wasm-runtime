package main

import (
	"fmt"
	"os"

	"github.com/kawabatas/toy-wasm-runtime/vm"
	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

func main() {
	data, err := os.ReadFile("./testdata/add.wasm")
	if err != nil {
		panic(err)
	}

	mod, err := wasm.DecodeModule(data)
	if err != nil {
		panic(err)
	}

	vm, err := vm.InstantiateModule(mod)
	if err != nil {
		panic(vm)
	}

	addParams := []uint64{40, 2}
	result, err := vm.InvokeFunction("add", addParams...)
	if err != nil {
		panic(err)
	}
	fmt.Println(result) // 42
}
