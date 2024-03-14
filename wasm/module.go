package wasm

// Magic is the 4 byte preamble (literally "\0asm") of the binary format
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-magic
var Magic = []byte{0x00, 0x61, 0x73, 0x6D}

// version is format version and doesn't change between known specification versions
var version = []byte{0x01, 0x00, 0x00, 0x00}

// SectionID identifies the sections of a Module in the WebAssembly 1.0 (20191205) Binary Format.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#sections%E2%91%A0
type SectionID = byte

const (
	// SectionIDCustom includes the standard defined NameSection and possibly others not defined in the standard.
	SectionIDCustom SectionID = iota // don't add anything not in https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#sections%E2%91%A0
	SectionIDType
	SectionIDImport
	SectionIDFunction
	SectionIDTable
	SectionIDMemory
	SectionIDGlobal
	SectionIDExport
	SectionIDStart
	SectionIDElement
	SectionIDCode
	SectionIDData
)

// SectionIDName returns the canonical name of a module section.
// https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#sections%E2%91%A0
func SectionIDName(sectionID SectionID) string {
	switch sectionID {
	case SectionIDCustom:
		return "custom"
	case SectionIDType:
		return "type"
	case SectionIDImport:
		return "import"
	case SectionIDFunction:
		return "function"
	case SectionIDTable:
		return "table"
	case SectionIDMemory:
		return "memory"
	case SectionIDGlobal:
		return "global"
	case SectionIDExport:
		return "export"
	case SectionIDStart:
		return "start"
	case SectionIDElement:
		return "element"
	case SectionIDCode:
		return "code"
	case SectionIDData:
		return "data"
	}
	return "unknown"
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#modules%E2%91%A8
type Module struct {
	TypeSection   []FunctionType
	ImportSection []Import
	ImportFunctionCount,
	ImportGlobalCount,
	ImportMemoryCount,
	ImportTableCount Index
	FunctionSection []Index
	TableSection    []Table
	MemorySection   *Memory
	GlobalSection   []Global
	ExportSection   []Export
	Exports         map[string]*Export
	StartSection    *Index
	CodeSection     []Code
	DataSection     []DataSegment
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-import
type Import struct {
	Type         ExternType
	Module       string
	Name         string
	DescFunc     Index
	DescTable    Table
	DescMem      *Memory
	DescGlobal   GlobalType
	IndexPerType Index
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-index
type Index = uint32

// Table describes the limits of elements and its type in a table.
type Table struct {
	Min  uint32
	Max  *uint32
	Type RefType
}

// Memory describes the limits of pages (64KB) in a memory.
type Memory struct {
	Min, Cap, Max uint32
	IsMaxEncoded  bool
	IsShared      bool
}

// Validate ensures values assigned to Min, Cap and Max are within valid thresholds.
func (m *Memory) Validate(memoryLimitPages uint32) error {
	// TODO
	return nil
}

type Global struct {
	Type GlobalType
	Init ConstantExpression
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-export
type Export struct {
	Type  ExternType
	Name  string
	Index Index
}

// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-code
type Code struct {
	LocalTypes              []ValueType
	Body                    []byte
	BodyOffsetInCodeSection uint64
}

type DataSegment struct {
	OffsetExpression ConstantExpression
	Init             []byte
	Passive          bool
}
