package main

import (
	"fmt"
	"os"

	"github.com/kawabatas/toy-wasm-runtime/vm"
	"github.com/kawabatas/toy-wasm-runtime/wasm"
)

func main() {
	data, err := os.ReadFile("./testdata/fib.wasm")
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

	for i := uint64(0); i <= 10; i++ {
		result, err := vm.InvokeFunction("fib", []uint64{i}...)
		if err != nil {
			panic(err)
		}
		fmt.Println(result)
		// 0
		// 1
		// 1
		// 2
		// 3
		// 5
		// 8
		// 13
		// 21
		// 34
		// 55
	}
}
