// This file is part of GoRE.
//
// Copyright (C) 2019-2024 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"reflect"
	"strings"
)

func typeDef(b *bytes.Buffer, st reflect.Type, bits int) {
	typeName := "uint64"
	if bits == 32 {
		typeName = "uint32"
	}

	_, _ = fmt.Fprintf(b, "type %s%d struct {\n", st.Name(), bits)

	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		fieldName := strings.ToUpper(field.Name[:1]) + field.Name[1:]
		t := field.Type.Kind()
		switch t {
		case reflect.Uintptr:
			_, _ = fmt.Fprintf(b, "%s %s\n", fieldName, typeName)
		case reflect.String:
			_, _ = fmt.Fprintf(b, "%s, %[1]slen %s\n", fieldName, typeName)
		case reflect.Pointer:
			_, _ = fmt.Fprintf(b, "%s %s\n", fieldName, typeName)
		case reflect.Slice:
			_, _ = fmt.Fprintf(b, "%s, %[1]slen, %[1]scap %s\n", fieldName, typeName)

		default:
			panic(fmt.Sprintf("unhandled type: %+v", t))
		}
	}

	_, _ = fmt.Fprint(b, "}\n\n")
}

func toModuledata(b *bytes.Buffer, st reflect.Type, bits int) {
	_, _ = fmt.Fprintf(b, "func (md %s%d) toModuledata() moduledata {\n", st.Name(), bits)
	_, _ = fmt.Fprint(b, "return moduledata{\n")

	for _, names := range [][2]string{
		{"Text", "Text"},
		{"NoPtrData", "Noptrdata"},
		{"Data", "Data"},
		{"Bss", "Bss"},
		{"NoPtrBss", "Noptrbss"},
		{"Types", "Types"},
	} {
		modFieldE(b, st, bits, names[0], names[1])
	}

	for _, names := range [][2]string{
		{"Typelink", "Typelinks"},
		{"ITabLink", "Itablinks"},
		{"FuncTab", "Ftab"},
		{"PCLNTab", "Pclntable"},
	} {
		modFieldLen(b, st, bits, names[0], names[1])
	}

	modFieldVal(b, st, bits, "GoFunc", "Gofunc")

	_, _ = fmt.Fprint(b, "}\n}\n\n")
}

func modFieldE(b *bytes.Buffer, st reflect.Type, bits int, modName, parsedName string) {
	endName := "E" + strings.ToLower(parsedName)
	if _, ok := st.FieldByName(strings.ToLower(parsedName)); !ok {
		return
	}
	if bits == 32 {
		_, _ = fmt.Fprintf(b, "%sAddr: uint64(md.%[3]s),\n%[1]sLen: uint64(md.%s - md.%s),\n", modName, endName, parsedName)
	} else {
		_, _ = fmt.Fprintf(b, "%sAddr: md.%[3]s,\n%[1]sLen: md.%s - md.%s,\n", modName, endName, parsedName)
	}
}

func modFieldLen(b *bytes.Buffer, st reflect.Type, bits int, modName, parsedName string) {
	lenName := parsedName + "len"
	if _, ok := st.FieldByName(strings.ToLower(parsedName)); !ok {
		return
	}
	if bits == 32 {
		_, _ = fmt.Fprintf(b, "%sAddr: uint64(md.%s),\n%[1]sLen: uint64(md.%[3]s),\n", modName, parsedName, lenName)
	} else {
		_, _ = fmt.Fprintf(b, "%sAddr: md.%s,\n%[1]sLen: md.%[3]s,\n", modName, parsedName, lenName)
	}
}

func modFieldVal(b *bytes.Buffer, st reflect.Type, bits int, modName, parsedName string) {
	if _, ok := st.FieldByName(strings.ToLower(parsedName)); !ok {
		return
	}
	if bits == 32 {
		_, _ = fmt.Fprintf(b, "%sVal: uint64(md.%s),\n", modName, parsedName)
	} else {
		_, _ = fmt.Fprintf(b, "%sVal: md.%s,\n", modName, parsedName)
	}
}

func generateModuleData() {
	b := &bytes.Buffer{}
	b.WriteString(moduleDataHeader)

	for _, iface := range []any{
		moduledata20{},
		moduledata18{},
		moduledata16{},
		moduledata8{},
		moduledata7{},
		moduledata5{},
	} {
		o := reflect.TypeOf(iface)
		typeDef(b, o, 64)
		toModuledata(b, o, 64)
		typeDef(b, o, 32)
		toModuledata(b, o, 32)
	}

	out, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(moduleDataOutputFile, out, 0o666)
	if err != nil {
		panic(err)
	}
}

/*
	Internal module structures from Go's runtime.
	TODO: auto extract from golang source runtime package.
*/

// Moduledata structure for Go 1.20 and newer (at least up to the last field covered here)

type moduledata20 struct {
	pcHeader     *pcHeader
	funcnametab  []byte
	cutab        []uint32
	filetab      []byte
	pctab        []byte
	pclntable    []byte
	ftab         []functab
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	covctrs, ecovctrs     uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr
	gofunc                uintptr // go.func.*

	textsectmap []textsect
	typelinks   []int32 // offsets from types
	itablinks   []*itab
}

// Moduledata structure for Go 1.18 and Go 1.19

type moduledata18 struct {
	pcHeader     *pcHeader
	funcnametab  []byte
	cutab        []uint32
	filetab      []byte
	pctab        []byte
	pclntable    []byte
	ftab         []functab
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr
	gofunc                uintptr // go.func.*

	textsectmap []textsect
	typelinks   []int32 // offsets from types
	itablinks   []*itab
}

// Moduledata structure for Go 1.16 to 1.17

type moduledata16 struct {
	pcHeader     *pcHeader
	funcnametab  []byte
	cutab        []uint32
	filetab      []byte
	pctab        []byte
	pclntable    []byte
	ftab         []functab
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr

	textsectmap []textsect
	typelinks   []int32 // offsets from types
	itablinks   []*itab
}

// Moduledata structure for Go 1.8 to 1.15

type moduledata8 struct {
	pclntable    []byte
	ftab         []functab
	filetab      []uint32
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr

	textsectmap []textsect
	typelinks   []int32 // offsets from types
	itablinks   []*itab
}

// Moduledata structure for Go 1.7

type moduledata7 struct {
	pclntable    []byte
	ftab         []functab
	filetab      []uint32
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr

	typelinks []int32 // offsets from types
	itablinks []*itab
}

// Moduledata structure for Go 1.5 to 1.6

type moduledata5 struct {
	pclntable    []byte
	ftab         []functab
	filetab      []uint32
	findfunctab  uintptr
	minpc, maxpc uintptr

	text, etext           uintptr
	noptrdata, enoptrdata uintptr
	data, edata           uintptr
	bss, ebss             uintptr
	noptrbss, enoptrbss   uintptr
	end, gcdata, gcbss    uintptr

	typelinks []*_type
}

// dummy definitions
type initTask struct{}
type pcHeader struct{}
type functab struct{}
type textsect struct{}
type itab struct{}
type ptabEntry struct{}
type modulehash struct{}
type _type struct{}
