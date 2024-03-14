package vm

type Frame struct {
	PC         uint64
	Function   *WasmFunction
	Locals     []uint64
	LabelStack *LabelStack
}

func NewFrame(f *WasmFunction, locals []uint64) *Frame {
	return &Frame{
		PC:         0,
		Function:   f,
		Locals:     locals,
		LabelStack: NewLabelStack(),
	}
}
