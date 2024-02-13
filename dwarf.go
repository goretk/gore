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

package gore

import (
	"bytes"
	"debug/dwarf"
	"encoding/binary"
)

const (
	// official DWARF language ID for Go
	// https://dwarfstd.org/languages.html
	dwLangGo int64 = 0x0016

	// DWARF operation; used to encode type offsets
	dwOpAddr = 0x03
)

func getGoRootFromDwarf(fh fileHandler) (string, bool) {
	return getDwarfString(fh, getDwarfStringCheck("runtime.defaultGOROOT"))
}

func getBuildVersionFromDwarf(fh fileHandler) (string, bool) {
	return getDwarfString(fh, getDwarfStringCheck("runtime.buildVersion"))
}

// DWARF entry plus any associated children
type dwarfEntryPlus struct {
	entry    *dwarf.Entry
	children []*dwarfEntryPlus
}

type dwarfwalkStatus uint8

const (
	dwStop dwarfwalkStatus = iota + 1
	dwContinue
	dwFound
)

func getDwarfString(fh fileHandler, check func(fh fileHandler, entry *dwarfEntryPlus) (string, dwarfwalkStatus)) (string, bool) {
	data, err := fh.getDwarf()
	if err != nil {
		return "", false
	}

	r := data.Reader()
	// walk through compilation units
getValOuter:
	for cu := dwarfReadEntry(r); cu != nil; cu = dwarfReadEntry(r) {
		if langField := cu.entry.AttrField(dwarf.AttrLanguage); langField == nil || langField.Val != dwLangGo {
			continue
		}
	getValInner:
		for _, entry := range cu.children {
			ret, status := check(fh, entry)
			switch status {
			case dwStop:
				break getValOuter
			case dwFound:
				return ret, true
			case dwContinue:
				continue getValInner
			}
		}
	}
	return "", false
}

// get, by name, a DWARF entry corresponding to a string constant
func getDwarfStringCheck(name string) func(fh fileHandler, entry *dwarfEntryPlus) (string, dwarfwalkStatus) {
	return func(fh fileHandler, d *dwarfEntryPlus) (string, dwarfwalkStatus) {
		entry := d.entry
		nameField := entry.AttrField(dwarf.AttrName)
		if nameField == nil {
			return "", dwContinue
		}

		if fieldName := nameField.Val.(string); fieldName != name {
			return "", dwContinue
		}

		return commonStringCheck(fh, entry)
	}
}

func commonStringCheck(fh fileHandler, entry *dwarf.Entry) (string, dwarfwalkStatus) {
	locationField := entry.AttrField(dwarf.AttrLocation)
	if locationField == nil {
		// unexpected failure
		return "", dwStop
	}
	location := locationField.Val.([]byte)
	// DWARF address operation followed by the machine byte order encoded address
	if location[0] != dwOpAddr {
		return "", dwStop
	}
	var addr uint64
	if fh.getFileInfo().WordSize == intSize32 {
		addr = uint64(fh.getFileInfo().ByteOrder.Uint32(location[1:]))
	} else {
		addr = fh.getFileInfo().ByteOrder.Uint64(location[1:])
	}

	sectionBase, data, err := fh.getSectionDataFromAddress(addr)
	if err != nil {
		return "", dwStop
	}
	off := addr - sectionBase
	r := bytes.NewReader(data[off:])
	var stringData [2]uint64
	if fh.getFileInfo().WordSize == intSize32 {
		var stringData32 [2]uint32
		err = binary.Read(r, fh.getFileInfo().ByteOrder, &stringData32)
		if err != nil {
			return "", dwStop
		}
		stringData[0] = uint64(stringData32[0])
		stringData[1] = uint64(stringData32[1])
	} else {
		err = binary.Read(r, fh.getFileInfo().ByteOrder, &stringData)
		if err != nil {
			return "", dwStop
		}
	}
	addr = stringData[0]
	stringLen := stringData[1]
	sectionBase, data, err = fh.getSectionDataFromAddress(addr)
	if err != nil {
		return "", dwStop
	}
	off = addr - sectionBase
	raw := data[off : off+stringLen]
	return string(raw), dwFound
}

func dwarfReadEntry(r *dwarf.Reader) *dwarfEntryPlus {
	entry, _ := r.Next()
	if entry == nil {
		return nil
	}
	var children []*dwarfEntryPlus
	if entry.Children {
		children = dwarfReadChildren(r)
	}
	return &dwarfEntryPlus{
		entry:    entry,
		children: children,
	}
}

func dwarfReadChildren(r *dwarf.Reader) []*dwarfEntryPlus {
	var ret []*dwarfEntryPlus

	for {
		e := dwarfReadEntry(r)
		if e.entry.Tag == 0 {
			return ret
		}
		ret = append(ret, e)
	}
}
