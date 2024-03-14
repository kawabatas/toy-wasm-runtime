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
