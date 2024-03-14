package wasm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"unicode/utf8"
	"unsafe"
)

// DecodeModule implements wasm.DecodeModule for the WebAssembly 1.0 (20191205) Binary Format
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-format%E2%91%A0
func DecodeModule(
	binary []byte,
) (*Module, error) {
	r := bytes.NewReader(binary)

	// Magic number.
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil || !bytes.Equal(buf, Magic) {
		return nil, ErrInvalidMagicNumber
	}

	// Version.
	if _, err := io.ReadFull(r, buf); err != nil || !bytes.Equal(buf, version) {
		return nil, ErrInvalidVersion
	}

	memoryLimitPages := MemoryLimitPages

	m := &Module{}
	for {
		// TODO: except custom sections, all others are required to be in order, but we aren't checking yet.
		// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#modules%E2%91%A0%E2%93%AA
		sectionID, err := r.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("read section id: %w", err)
		}

		sectionSize, _, err := DecodeUint32(r)
		if err != nil {
			return nil, fmt.Errorf("get size of section %s: %v", SectionIDName(sectionID), err)
		}

		sectionContentStart := r.Len()
		switch sectionID {
		case SectionIDCustom:
			err = errors.New("not implemented custom section")
		case SectionIDType:
			m.TypeSection, err = decodeTypeSection(r)
		case SectionIDImport:
			m.ImportSection, m.ImportFunctionCount, m.ImportGlobalCount, m.ImportMemoryCount, m.ImportTableCount, err = decodeImportSection(r)
			if err != nil {
				return nil, err // avoid re-wrapping the error.
			}
		case SectionIDFunction:
			m.FunctionSection, err = decodeFunctionSection(r)
		case SectionIDTable:
			err = errors.New("not implemented table section")
		case SectionIDMemory:
			m.MemorySection, err = decodeMemorySection(r, memoryLimitPages)
		case SectionIDGlobal:
			err = errors.New("not implemented global section")
		case SectionIDExport:
			m.ExportSection, m.Exports, err = decodeExportSection(r)
		case SectionIDStart:
			if m.StartSection != nil {
				return nil, errors.New("multiple start sections are invalid")
			}
			m.StartSection, err = decodeStartSection(r)
		case SectionIDElement:
			err = errors.New("not implemented element section")
		case SectionIDCode:
			m.CodeSection, err = decodeCodeSection(r)
		case SectionIDData:
			m.DataSection, err = decodeDataSection(r)
		default:
			err = ErrInvalidSectionID
		}

		readBytes := sectionContentStart - r.Len()
		if err == nil && int(sectionSize) != readBytes {
			err = fmt.Errorf("invalid section length: expected to be %d but got %d", sectionSize, readBytes)
		}

		if err != nil {
			return nil, fmt.Errorf("section %s: %v", SectionIDName(sectionID), err)
		}
	}

	return m, nil
}

func decodeTypeSection(r *bytes.Reader) ([]FunctionType, error) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("get size of vector: %w", err)
	}

	result := make([]FunctionType, vs)
	for i := uint32(0); i < vs; i++ {
		if err = decodeFunctionType(r, &result[i]); err != nil {
			return nil, fmt.Errorf("read %d-th type: %v", i, err)
		}
	}
	return result, nil
}

func decodeImportSection(
	r *bytes.Reader,
) (result []Import,
	funcCount, globalCount, memoryCount, tableCount Index, err error,
) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		err = fmt.Errorf("get size of vector: %w", err)
		return
	}

	result = make([]Import, vs)
	for i := uint32(0); i < vs; i++ {
		imp := &result[i]
		if err = decodeImport(r, i, imp); err != nil {
			return
		}
		switch imp.Type {
		case ExternTypeFunc:
			imp.IndexPerType = funcCount
			funcCount++
		case ExternTypeGlobal:
			imp.IndexPerType = globalCount
			globalCount++
		case ExternTypeMemory:
			imp.IndexPerType = memoryCount
			memoryCount++
		case ExternTypeTable:
			imp.IndexPerType = tableCount
			tableCount++
		}
	}
	return
}

func decodeFunctionSection(r *bytes.Reader) ([]uint32, error) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("get size of vector: %w", err)
	}

	result := make([]uint32, vs)
	for i := uint32(0); i < vs; i++ {
		if result[i], _, err = DecodeUint32(r); err != nil {
			return nil, fmt.Errorf("get type index: %w", err)
		}
	}
	return result, err
}

func decodeMemorySection(
	r *bytes.Reader,
	memoryLimitPages uint32,
) (*Memory, error) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("error reading size")
	}
	if vs > 1 {
		return nil, fmt.Errorf("at most one memory allowed in module, but read %d", vs)
	} else if vs == 0 {
		// memory count can be zero.
		return nil, nil
	}

	return decodeMemory(r, memoryLimitPages)
}

func decodeExportSection(r *bytes.Reader) ([]Export, map[string]*Export, error) {
	vs, _, sizeErr := DecodeUint32(r)
	if sizeErr != nil {
		return nil, nil, fmt.Errorf("get size of vector: %v", sizeErr)
	}

	exportMap := make(map[string]*Export, vs)
	exportSection := make([]Export, vs)
	for i := Index(0); i < vs; i++ {
		export := &exportSection[i]
		err := decodeExport(r, export)
		if err != nil {
			return nil, nil, fmt.Errorf("read export: %w", err)
		}
		if _, ok := exportMap[export.Name]; ok {
			return nil, nil, fmt.Errorf("export[%d] duplicates name %q", i, export.Name)
		} else {
			exportMap[export.Name] = export
		}
	}
	return exportSection, exportMap, nil
}

func decodeStartSection(r *bytes.Reader) (*Index, error) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("get function index: %w", err)
	}
	return &vs, nil
}

func decodeCodeSection(r *bytes.Reader) ([]Code, error) {
	codeSectionStart := uint64(r.Len())
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("get size of vector: %w", err)
	}

	result := make([]Code, vs)
	for i := uint32(0); i < vs; i++ {
		err = decodeCode(r, codeSectionStart, &result[i])
		if err != nil {
			return nil, fmt.Errorf("read %d-th code segment: %v", i, err)
		}
	}
	return result, nil
}

func decodeDataSection(r *bytes.Reader) ([]DataSegment, error) {
	vs, _, err := DecodeUint32(r)
	if err != nil {
		return nil, fmt.Errorf("get size of vector: %w", err)
	}

	result := make([]DataSegment, vs)
	for i := uint32(0); i < vs; i++ {
		if err = decodeDataSegment(r, &result[i]); err != nil {
			return nil, fmt.Errorf("read data segment: %w", err)
		}
	}
	return result, nil
}

// ==========================================================================
// ==========================================================================
// ==========================================================================

func decodeImport(
	r *bytes.Reader,
	idx uint32,
	ret *Import,
) (err error) {
	if ret.Module, _, err = decodeUTF8(r, "import module"); err != nil {
		err = fmt.Errorf("import[%d] error decoding module: %w", idx, err)
		return
	}

	if ret.Name, _, err = decodeUTF8(r, "import name"); err != nil {
		err = fmt.Errorf("import[%d] error decoding name: %w", idx, err)
		return
	}

	b, err := r.ReadByte()
	if err != nil {
		err = fmt.Errorf("import[%d] error decoding type: %w", idx, err)
		return
	}
	ret.Type = b
	switch ret.Type {
	case ExternTypeFunc:
		ret.DescFunc, _, err = DecodeUint32(r)
	case ExternTypeTable:
		err = errors.New("not implemented import table")
	case ExternTypeMemory:
		err = errors.New("not implemented import memory")
	case ExternTypeGlobal:
		err = errors.New("not implemented import global")
	default:
		err = fmt.Errorf("%w: invalid byte for importdesc: %#x", ErrInvalidByte, b)
	}
	if err != nil {
		err = fmt.Errorf("import[%d] %s[%s.%s]: %w", idx, ExternTypeName(ret.Type), ret.Module, ret.Name, err)
	}
	return
}

func decodeMemory(
	r *bytes.Reader,
	memoryLimitPages uint32,
) (*Memory, error) {
	min, maxP, shared, err := decodeLimitsType(r)
	if err != nil {
		return nil, err
	}

	capacity, max := memoryLimitPages, memoryLimitPages
	mem := &Memory{Min: min, Cap: capacity, Max: max, IsMaxEncoded: maxP != nil, IsShared: shared}

	return mem, mem.Validate(memoryLimitPages)
}

func decodeExport(r *bytes.Reader, ret *Export) (err error) {
	if ret.Name, _, err = decodeUTF8(r, "export name"); err != nil {
		return
	}

	b, err := r.ReadByte()
	if err != nil {
		err = fmt.Errorf("error decoding export kind: %w", err)
		return
	}

	ret.Type = b
	switch ret.Type {
	case ExternTypeFunc, ExternTypeTable, ExternTypeMemory, ExternTypeGlobal:
		if ret.Index, _, err = DecodeUint32(r); err != nil {
			err = fmt.Errorf("error decoding export index: %w", err)
		}
	default:
		err = fmt.Errorf("%w: invalid byte for exportdesc: %#x", ErrInvalidByte, b)
	}
	return
}

func decodeCode(r *bytes.Reader, codeSectionStart uint64, ret *Code) (err error) {
	ss, _, err := DecodeUint32(r)
	if err != nil {
		return fmt.Errorf("get the size of code: %w", err)
	}
	remaining := int64(ss)

	// Parse #locals.
	ls, bytesRead, err := DecodeUint32(r)
	remaining -= int64(bytesRead)
	if err != nil {
		return fmt.Errorf("get the size locals: %v", err)
	} else if remaining < 0 {
		return io.EOF
	}

	// Validate the locals.
	bytesRead = 0
	var sum uint64
	for i := uint32(0); i < ls; i++ {
		num, n, err := DecodeUint32(r)
		if err != nil {
			return fmt.Errorf("read n of locals: %v", err)
		} else if remaining < 0 {
			return io.EOF
		}

		sum += uint64(num)

		b, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("read type of local: %v", err)
		}

		bytesRead += n + 1
		switch vt := b; vt {
		case ValueTypeI32, ValueTypeF32, ValueTypeI64, ValueTypeF64,
			ValueTypeFuncref, ValueTypeExternref, ValueTypeV128:
		default:
			return fmt.Errorf("invalid local type: 0x%x", vt)
		}
	}

	if sum > math.MaxUint32 {
		return fmt.Errorf("too many locals: %d", sum)
	}

	// Rewind the buffer.
	_, err = r.Seek(-int64(bytesRead), io.SeekCurrent)
	if err != nil {
		return err
	}

	localTypes := make([]ValueType, 0, sum)
	for i := uint32(0); i < ls; i++ {
		num, bytesRead, err := DecodeUint32(r)
		remaining -= int64(bytesRead) + 1 // +1 for the subsequent ReadByte
		if err != nil {
			return fmt.Errorf("read n of locals: %v", err)
		} else if remaining < 0 {
			return io.EOF
		}

		b, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("read type of local: %v", err)
		}

		for j := uint32(0); j < num; j++ {
			localTypes = append(localTypes, b)
		}
	}

	bodyOffsetInCodeSection := codeSectionStart - uint64(r.Len())
	body := make([]byte, remaining)
	if _, err = io.ReadFull(r, body); err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if endIndex := len(body) - 1; endIndex < 0 || body[endIndex] != OpcodeEnd {
		return fmt.Errorf("expr not end with OpcodeEnd")
	}

	ret.BodyOffsetInCodeSection = bodyOffsetInCodeSection
	ret.LocalTypes = localTypes
	ret.Body = body
	return nil
}

// dataSegmentPrefix represents three types of data segments.
// https://www.w3.org/TR/2022/WD-wasm-core-2-20220419/binary/modules.html#data-section
type dataSegmentPrefix = uint32

const (
	dataSegmentPrefixActive                dataSegmentPrefix = 0x0
	dataSegmentPrefixPassive               dataSegmentPrefix = 0x1
	dataSegmentPrefixActiveWithMemoryIndex dataSegmentPrefix = 0x2
)

func decodeDataSegment(r *bytes.Reader, ret *DataSegment) (err error) {
	dataSegmentPrefx, _, err := DecodeUint32(r)
	if err != nil {
		err = fmt.Errorf("read data segment prefix: %w", err)
		return
	}

	switch dataSegmentPrefx {
	case dataSegmentPrefixActive,
		dataSegmentPrefixActiveWithMemoryIndex:

		err = decodeConstantExpression(r, &ret.OffsetExpression)
		if err != nil {
			return fmt.Errorf("read offset expression: %v", err)
		}
	default:
		err = fmt.Errorf("invalid data segment prefix: 0x%x", dataSegmentPrefx)
		return
	}

	vs, _, err := DecodeUint32(r)
	if err != nil {
		err = fmt.Errorf("get the size of vector: %v", err)
		return
	}

	ret.Init = make([]byte, vs)
	if _, err = io.ReadFull(r, ret.Init); err != nil {
		err = fmt.Errorf("read bytes for init: %v", err)
	}
	return
}

// ==========================================================================
// ==========================================================================
// ==========================================================================

func decodeValueTypes(r *bytes.Reader, num uint32) ([]ValueType, error) {
	if num == 0 {
		return nil, nil
	}

	ret := make([]ValueType, num)
	_, err := io.ReadFull(r, ret)
	if err != nil {
		return nil, err
	}

	for _, v := range ret {
		switch v {
		case ValueTypeI32, ValueTypeF32, ValueTypeI64, ValueTypeF64,
			ValueTypeExternref, ValueTypeFuncref, ValueTypeV128:
		default:
			return nil, fmt.Errorf("invalid value type: %d", v)
		}
	}
	return ret, nil
}

func decodeFunctionType(r *bytes.Reader, ret *FunctionType) (err error) {
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("read leading byte: %w", err)
	}

	if b != 0x60 {
		return fmt.Errorf("%w: %#x != 0x60", ErrInvalidByte, b)
	}

	paramCount, _, err := DecodeUint32(r)
	if err != nil {
		return fmt.Errorf("could not read parameter count: %w", err)
	}

	paramTypes, err := decodeValueTypes(r, paramCount)
	if err != nil {
		return fmt.Errorf("could not read parameter types: %w", err)
	}

	resultCount, _, err := DecodeUint32(r)
	if err != nil {
		return fmt.Errorf("could not read result count: %w", err)
	}

	resultTypes, err := decodeValueTypes(r, resultCount)
	if err != nil {
		return fmt.Errorf("could not read result types: %w", err)
	}

	ret.Params = paramTypes
	ret.Results = resultTypes

	// cache the key for the function type
	_ = ret.String()

	return nil
}

// decodeLimitsType returns the `limitsType` (min, max) decoded with the WebAssembly 1.0 (20191205) Binary Format.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#limits%E2%91%A6
func decodeLimitsType(r *bytes.Reader) (min uint32, max *uint32, shared bool, err error) {
	var flag byte
	if flag, err = r.ReadByte(); err != nil {
		err = fmt.Errorf("read leading byte: %v", err)
		return
	}

	switch flag {
	case 0x00, 0x02:
		min, _, err = DecodeUint32(r)
		if err != nil {
			err = fmt.Errorf("read min of limit: %v", err)
		}
	case 0x01, 0x03:
		min, _, err = DecodeUint32(r)
		if err != nil {
			err = fmt.Errorf("read min of limit: %v", err)
			return
		}
		var m uint32
		if m, _, err = DecodeUint32(r); err != nil {
			err = fmt.Errorf("read max of limit: %v", err)
		} else {
			max = &m
		}
	default:
		err = fmt.Errorf("%v for limits: %#x not in (0x00, 0x01, 0x02, 0x03)", ErrInvalidByte, flag)
	}

	shared = flag == 0x02 || flag == 0x03

	return
}

func decodeConstantExpression(r *bytes.Reader, ret *ConstantExpression) error {
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("read opcode: %v", err)
	}

	remainingBeforeData := int64(r.Len())
	offsetAtData := r.Size() - remainingBeforeData

	opcode := b
	switch opcode {
	case OpcodeI32Const:
		// Treat constants as signed as their interpretation is not yet known per /RATIONALE.md
		_, _, err = DecodeInt32(r)
	default:
		return fmt.Errorf("%v for const expression opt code: %#x", ErrInvalidByte, b)
	}

	if err != nil {
		return fmt.Errorf("read value: %v", err)
	}

	if b, err = r.ReadByte(); err != nil {
		return fmt.Errorf("look for end opcode: %v", err)
	}

	if b != OpcodeEnd {
		return fmt.Errorf("constant expression has been not terminated")
	}

	ret.Data = make([]byte, remainingBeforeData-int64(r.Len())-1)
	if _, err = r.ReadAt(ret.Data, offsetAtData); err != nil {
		return fmt.Errorf("error re-buffering ConstantExpression.Data")
	}
	ret.Opcode = opcode
	return nil
}

// decodeUTF8 decodes a size prefixed string from the reader, returning it and the count of bytes read.
// contextFormat and contextArgs apply an error format when present
func decodeUTF8(r *bytes.Reader, contextFormat string, contextArgs ...interface{}) (string, uint32, error) {
	size, sizeOfSize, err := DecodeUint32(r)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read %s size: %w", fmt.Sprintf(contextFormat, contextArgs...), err)
	}

	if size == 0 {
		return "", uint32(sizeOfSize), nil
	}

	buf := make([]byte, size)
	if _, err = io.ReadFull(r, buf); err != nil {
		return "", 0, fmt.Errorf("failed to read %s: %w", fmt.Sprintf(contextFormat, contextArgs...), err)
	}

	if !utf8.Valid(buf) {
		return "", 0, fmt.Errorf("%s is not valid UTF-8", fmt.Sprintf(contextFormat, contextArgs...))
	}

	// TODO: use unsafe.String after flooring Go 1.20.
	ret := *(*string)(unsafe.Pointer(&buf))
	return ret, size + uint32(sizeOfSize), nil
}
