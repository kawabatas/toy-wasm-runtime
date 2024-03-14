package vm

import (
	"encoding/binary"
	"fmt"
	"os"
)

const wasiPreview1 = "wasi_snapshot_preview1"

type wasiFdWrite struct {
	vm *VM
}

func newWasiFdWrite(vm *VM) *wasiFdWrite {
	return &wasiFdWrite{
		vm: vm,
	}
}

// fdWrite is the WASI function named FdWriteName which writes to a file
// descriptor.
//
// # Parameters
//
//   - fd: an opened file descriptor to write data to
//   - iovs: offset in api.Memory to read offset, size pairs representing the
//     data to write to `fd`
//   - Both offset and length are encoded as uint32le.
//   - iovsCount: count of memory offset, size pairs to read sequentially
//     starting at iovs
//   - resultNwritten: offset in api.Memory to write the number of bytes
//     written
//
// Result (Errno)
//
// The return value is 0 except the following error conditions:
//   - sys.EBADF: `fd` is invalid
//   - sys.EFAULT: `iovs` or `resultNwritten` point to an offset out of memory
//   - sys.EIO: a file system error
//
// For example, this function needs to first read `iovs` to determine what to
// write to `fd`. If parameters iovs=1 iovsCount=2, this function reads two
// offset/length pairs from api.Memory:
//
//	                  iovs[0]                  iovs[1]
//	          +---------------------+   +--------------------+
//	          | uint32le    uint32le|   |uint32le    uint32le|
//	          +---------+  +--------+   +--------+  +--------+
//	          |         |  |        |   |        |  |        |
//	[]byte{?, 18, 0, 0, 0, 4, 0, 0, 0, 23, 0, 0, 0, 2, 0, 0, 0, ?... }
//	   iovs --^            ^            ^           ^
//	          |            |            |           |
//	 offset --+   length --+   offset --+  length --+
//
// This function reads those chunks api.Memory into the `fd` sequentially.
//
//	                    iovs[0].length        iovs[1].length
//	                   +--------------+       +----+
//	                   |              |       |    |
//	[]byte{ 0..16, ?, 'w', 'a', 'z', 'e', ?, 'r', 'o', ? }
//	  iovs[0].offset --^                      ^
//	                         iovs[1].offset --+
//
// Since "wazero" was written, if parameter resultNwritten=26, this function
// writes the below to api.Memory:
//
//	                   uint32le
//	                  +--------+
//	                  |        |
//	[]byte{ 0..24, ?, 6, 0, 0, 0', ? }
//	 resultNwritten --^
//
// Note: This is similar to `writev` in POSIX. https://linux.die.net/man/3/writev
//
// See fdRead
// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#ciovec
// and https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_write
func (f *wasiFdWrite) Call(fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	// fd          => file_descriptor - 1 for stdout
	// iovsPtr     => *iovs - The pointer to the iov array, which is stored at memory location 0
	// iovsLen     => iovs_len - We're printing 1 string stored in an iov - so one.
	// nwrittenPtr => nwritten - A place in memory to store the number of bytes written
	if fd != 1 {
		panic(fmt.Errorf("invalid file descriptor: %d", fd))
	}

	mem := f.vm.Store.Memory
	var nwritten uint32
	for i := int32(0); i < iovsLen; i++ {
		iovPtr := iovsPtr + i*8 // Size 8
		offset := binary.LittleEndian.Uint32(mem[iovPtr:])
		l := binary.LittleEndian.Uint32(mem[iovPtr+4:])
		n, err := os.Stdout.Write(mem[offset : offset+l])
		if err != nil {
			panic(err)
		}
		nwritten += uint32(n)
	}
	binary.LittleEndian.PutUint32(mem[nwrittenPtr:], nwritten)
	return 0
}
