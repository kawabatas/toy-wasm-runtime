package vm

const stackSize = 128

type Stack struct {
	stack []uint64
	sp    int
}

func NewStack() *Stack {
	return &Stack{
		stack: make([]uint64, stackSize),
		sp:    -1,
	}
}

func (s *Stack) Pop() uint64 {
	ret := s.stack[s.sp]
	s.sp--
	return ret
}

func (s *Stack) Drop() {
	s.sp--
}

func (s *Stack) Peek() uint64 {
	return s.stack[s.sp]
}

func (s *Stack) Push(val uint64) {
	// TODO: FIXME: overflow StackSize
	s.stack[s.sp+1] = val
	s.sp++
}

type LabelStack struct {
	Stack []*Label
	SP    int
}

type Label struct {
	Arity          int
	ContinuationPC uint64
	EndPC          uint64
}

func NewLabelStack() *LabelStack {
	return &LabelStack{
		Stack: make([]*Label, stackSize),
		SP:    -1,
	}
}

func (s *LabelStack) Pop() *Label {
	ret := s.Stack[s.SP]
	s.SP--
	return ret
}

func (s *LabelStack) Push(val *Label) {
	s.Stack[s.SP+1] = val
	s.SP++
}
