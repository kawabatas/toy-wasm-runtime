package wasm

// See
//
//	https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#control-instructions%E2%91%A6
//	https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#a7-index-of-instructions
type Opcode = byte

const (
	OpcodeUnreachable Opcode = 0x00
	OpcodeNop         Opcode = 0x01
	OpcodeEnd         Opcode = 0x0b
	OpcodeReturn      Opcode = 0x0f
	OpcodeCall        Opcode = 0x10
	OpcodeDrop        Opcode = 0x1a
	OpcodeI32Load     Opcode = 0x28
	OpcodeI32Store    Opcode = 0x36
	OpcodeI32Const    Opcode = 0x41
)

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-expr
type ConstantExpression struct {
	Opcode Opcode
	Data   []byte
}
