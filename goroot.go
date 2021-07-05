// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"golang.org/x/arch/x86/x86asm"
	"reflect"
	"unicode/utf8"
)

func tryFromGOROOT(f *GoFile) (string, error) {
	// Check for non supported architectures.
	if f.FileInfo.Arch != Arch386 && f.FileInfo.Arch != ArchAMD64 {
		return "", nil
	}

	is32 := false
	if f.FileInfo.Arch == Arch386 {
		is32 = true
	}

	// Find runtime.GOROOT function.
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
		return "", ErrNoGoRootFound
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

		// Check if it's a "mov" instruction.
		if inst.Op != x86asm.MOV {
			continue
		}
		if inst.Args[0] != x86asm.RAX && inst.Args[0] != x86asm.EAX {
			continue
		}
		arg := inst.Args[1].(x86asm.Mem)

		// First assume that the address is a direct addressing.
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		} else if arg.Base == 0 && arg.Disp > 0 {
			// In order to support x32 direct addressing
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
		if !utf8.ValidString(ver) {
			return "", ErrNoGoRootFound
		}
		return ver, nil
	}

	// for go version vary from 1.5 to 1.9
	s = 0
	for s < len(buf) {
		var leaInst, movInst, movInst2, addInst, retInst x86asm.Inst
		var err error
		// We must find an instruction set of the form

		//.text:00405DB3                 lea     eax, loc_4A5B32          // goroot string
		//.text:00405DB9                 mov     [esp+10h+_r0.str], eax
		//.text:00405DBD                 mov     [esp+10h+_r0.len], 0Dh   // goroot length
		//.text:00405DC5                 add     esp, 10h
		//.text:00405DC8                 retn

		leaInst, err = x86asm.Decode(buf[s:], mode)
		if err != nil {
			return "", nil
		}
		s = s + leaInst.Len
		if leaInst.Op != x86asm.LEA {
			continue
		}
		arg := leaInst.Args[1].(x86asm.Mem)
		if arg.Base == x86asm.ESP || arg.Base == x86asm.RSP {
			continue
		}
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		} else if arg.Base == 0 && arg.Disp > 0 {
			// In order to support x32 direct addressing
		} else {
			continue
		}
		movInst, err = x86asm.Decode(buf[s:], mode)
		if err != nil {
			return "", nil
		}
		s = s + movInst.Len
		if movInst.Op != x86asm.MOV {
			continue
		}
		movInst2, err = x86asm.Decode(buf[s:], mode)
		if err != nil {
			return "", nil
		}
		s = s + movInst2.Len
		if movInst2.Op != x86asm.MOV {
			continue
		}
		addInst, err = x86asm.Decode(buf[s:], mode)
		if err != nil {
			return "", nil
		}
		s = s + addInst.Len
		if addInst.Op != x86asm.ADD {
			continue
		}
		retInst, err = x86asm.Decode(buf[s:], mode)
		if err != nil {
			return "", nil
		}
		s = s + retInst.Len
		if retInst.Op != x86asm.RET {
			continue
		}
		length := movInst2.Args[1].(x86asm.Imm)
		bstr, _ := f.Bytes(uint64(addr), uint64(length))
		if bstr == nil {
			continue
		}
		ver := string(bstr)
		if !utf8.ValidString(ver) {
			return "", ErrNoGoRootFound
		}
		return ver, nil
	}

	return "", ErrNoGoRootFound
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

	// Find time.init function.
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
		return "", ErrNoGoRootFound
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

		// Check if it's a "mov" instruction.
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

		// First assume that the address is a direct addressing.
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		} else if arg.Base == 0 && arg.Disp > 0 {
			// In order to support x32 direct addressing
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
		if !utf8.ValidString(ver) {
			return "", ErrNoGoRootFound
		}
		return ver, nil
	}
	return "", ErrNoGoRootFound
}

func findGoRootPath(f *GoFile) (string, error) {
	var goroot string
	// There is no GOROOT function may be inlined (after go1.16)
	// at this time GOROOT is obtained through time_init function
	goroot, err := tryFromGOROOT(f)
	if err != nil {
		if err == ErrNoGoRootFound {
			return tryFromTimeInit(f)
		}
		return "", err
	}
	return goroot, nil
}
