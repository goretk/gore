package gore

import (
	"bytes"
	"errors"
	"golang.org/x/arch/x86/x86asm"
	"reflect"
	"unicode"
)

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func tryFromGOROOT(f *GoFile) (string, error) {
	// Check for non supported architectures.
	if f.FileInfo.Arch != Arch386 && f.FileInfo.Arch != ArchAMD64 {
		return "", nil
	}

	is32 := false
	if f.FileInfo.Arch == Arch386 {
		is32 = true
	}

	// Find shedinit function.
	var fcn *Function
	std, err := f.GetSTDLib()
	if err != nil {
		return "", nil
	}

pkgLoop:
	for _, v := range std {
		if v.Name != "runtime" {
			continue
		}
		for _, vv := range v.Functions {
			if vv.Name != "GOROOT" {
				continue
			}
			fcn = vv
			break pkgLoop
		}
	}

	// Check if the functions was found
	if fcn == nil {
		// If we can't find the function there is nothing to do.
		return "", nil
	}
	// Get the raw hex.
	buf, err := f.Bytes(fcn.Offset, fcn.End-fcn.Offset)
	if err != nil {
		return "", nil
	}
	s := 0
	mode := f.FileInfo.WordSize * 8

	for s < len(buf) {
		inst, err := x86asm.Decode(buf[s:], mode)
		if err != nil {
			// If we fail to decode the instruction, something is wrong so
			// bailout.
			return "", nil
		}

		// Update next instruction location.
		s = s + inst.Len

		// Check if it's a "lea" instruction.
		if inst.Op != x86asm.MOV {
			continue
		}
		if inst.Args[0] != x86asm.RAX && inst.Args[0] != x86asm.EAX {
			continue
		}
		arg := inst.Args[1].(x86asm.Mem)
		// Check what it's loading and if it's pointing to the compiler version used.
		// First assume that the address is a direct addressing.
		//
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		} else if arg.Base == 0 && arg.Disp > 0 {
		} else {
			continue
		}
		// If the addressing is based on the stack pointer, this is not the right
		// instruction.
		if arg.Base == x86asm.ESP || arg.Base == x86asm.RSP {
			continue
		}

		// Resolve the pointer to the string. If we get no data, this is not the
		// right instruction.
		b, _ := f.Bytes(uint64(addr), uint64(0x20))
		if b == nil {
			continue
		}

		r := bytes.NewReader(b)
		ptr, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}
		l, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}

		bstr, _ := f.Bytes(ptr, l)
		if bstr == nil {
			continue
		}
		ver := string(bstr)
		if !isASCII(ver) {
			return "", nil
		}
		return ver, nil
	}
	return "", errors.New("not found GoRoot")
}

func tryFromTimeInit(f *GoFile) (string, error) {
	// Check for non supported architectures.
	if f.FileInfo.Arch != Arch386 && f.FileInfo.Arch != ArchAMD64 {
		return "", nil
	}

	is32 := false
	if f.FileInfo.Arch == Arch386 {
		is32 = true
	}

	// Find shedinit function.
	var fcn *Function
	std, err := f.GetSTDLib()
	if err != nil {
		return "", nil
	}

pkgLoop:
	for _, v := range std {
		if v.Name != "time" {
			continue
		}
		for _, vv := range v.Functions {
			if vv.Name != "init" {
				continue
			}
			fcn = vv
			break pkgLoop
		}
	}

	// Check if the functions was found
	if fcn == nil {
		// If we can't find the function there is nothing to do.
		return "", nil
	}
	// Get the raw hex.
	buf, err := f.Bytes(fcn.Offset, fcn.End-fcn.Offset)
	if err != nil {
		return "", nil
	}
	s := 0
	mode := f.FileInfo.WordSize * 8

	for s < len(buf) {
		inst, err := x86asm.Decode(buf[s:], mode)
		if err != nil {
			// If we fail to decode the instruction, something is wrong so
			// bailout.
			return "", nil
		}

		// Update next instruction location.
		s = s + inst.Len

		// Check if it's a "lea" instruction.
		if inst.Op != x86asm.MOV {
			continue
		}
		if inst.Args[0] != x86asm.RAX && inst.Args[0] != x86asm.ECX {
			continue
		}
		kindof := reflect.TypeOf(inst.Args[1])
		if kindof.String() != "x86asm.Mem" {
			continue
		}
		arg := inst.Args[1].(x86asm.Mem)
		// Check what it's loading and if it's pointing to the compiler version used.
		// First assume that the address is a direct addressing.
		//
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		} else if arg.Base == 0 && arg.Disp > 0 {
		} else {
			continue
		}
		// Resolve the pointer to the string. If we get no data, this is not the
		// right instruction.
		b, _ := f.Bytes(uint64(addr), uint64(0x20))
		if b == nil {
			continue
		}

		r := bytes.NewReader(b)
		ptr, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}
		l, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			// Probably not the right instruction, so go to next.
			continue
		}

		bstr, _ := f.Bytes(ptr, l)
		if bstr == nil {
			continue
		}
		ver := string(bstr)
		if !isASCII(ver) {
			return "", nil
		}
		return ver, nil
	}
	return "", errors.New("not found GoRoot")
}

func findGoRootPath(f *GoFile) (string, error) {
	fileInfo := f.FileInfo
	if GoVersionCompare("go1.16beta1", fileInfo.goversion.Name) >= 0 {
		return tryFromTimeInit(f)
	} else {
		return tryFromGOROOT(f)
	}
}
