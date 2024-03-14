package wasm

import "fmt"

// ValueType describes a parameter or result type mapped to a WebAssembly function signature.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-valtype
type ValueType = byte

const (
	ValueTypeI32       ValueType = 0x7f
	ValueTypeI64       ValueType = 0x7e
	ValueTypeF32       ValueType = 0x7d
	ValueTypeF64       ValueType = 0x7c
	ValueTypeExternref ValueType = 0x6f
	ValueTypeV128      ValueType = 0x7b
	ValueTypeFuncref   ValueType = 0x70
)

func ValueTypeName(t ValueType) string {
	switch t {
	case ValueTypeI32:
		return "i32"
	case ValueTypeI64:
		return "i64"
	case ValueTypeF32:
		return "f32"
	case ValueTypeF64:
		return "f64"
	case ValueTypeExternref:
		return "externref"
	case ValueTypeV128:
		return "v128"
	case ValueTypeFuncref:
		return "funcref"
	}
	return "unknown"
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#function-types%E2%91%A0
type FunctionType struct {
	Params            []ValueType
	Results           []ValueType
	string            string
	ParamNumInUint64  int
	ResultNumInUint64 int
}

func (f *FunctionType) String() string {
	return f.key()
}

func (f *FunctionType) key() string {
	if f.string != "" {
		return f.string
	}
	var ret string
	for _, b := range f.Params {
		ret += ValueTypeName(b)
	}
	if len(f.Params) == 0 {
		ret += "v_"
	} else {
		ret += "_"
	}
	for _, b := range f.Results {
		ret += ValueTypeName(b)
	}
	if len(f.Results) == 0 {
		ret += "v"
	}
	f.string = ret
	return ret
}

type GlobalType struct {
	ValType ValueType
	Mutable bool
}

// ExternType classifies imports and exports with their respective types.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#external-types%E2%91%A0
type ExternType = byte

const (
	ExternTypeFunc   ExternType = 0x00
	ExternTypeTable  ExternType = 0x01
	ExternTypeMemory ExternType = 0x02
	ExternTypeGlobal ExternType = 0x03
)

const (
	ExternTypeFuncName   = "func"
	ExternTypeTableName  = "table"
	ExternTypeMemoryName = "memory"
	ExternTypeGlobalName = "global"
)

func ExternTypeName(et ExternType) string {
	switch et {
	case ExternTypeFunc:
		return ExternTypeFuncName
	case ExternTypeTable:
		return ExternTypeTableName
	case ExternTypeMemory:
		return ExternTypeMemoryName
	case ExternTypeGlobal:
		return ExternTypeGlobalName
	}
	return fmt.Sprintf("%#x", et)
}

// RefType is either RefTypeFuncref or RefTypeExternref as of WebAssembly core 2.0.
type RefType = byte

const (
	RefTypeFuncref   = ValueTypeFuncref
	RefTypeExternref = ValueTypeExternref
)

func RefTypeName(t RefType) (ret string) {
	switch t {
	case RefTypeFuncref:
		ret = "funcref"
	case RefTypeExternref:
		ret = "externref"
	default:
		ret = fmt.Sprintf("unknown(0x%x)", t)
	}
	return
}
