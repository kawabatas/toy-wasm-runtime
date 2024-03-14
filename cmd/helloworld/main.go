package main

import (
	"os"

	"github.com/kawabatas/toy-wasm-runtime/vm"
	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

func main() {
	data, err := os.ReadFile("./testdata/helloworld.wasm")
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

	if err := vm.InvokeFunction("_start"); err != nil {
		panic(err)
	}
}
