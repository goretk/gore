// Copyright (C) 2026 GoRE Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package gore

import (
	"errors"
	"fmt"
	"reflect"
)

// ITabLinksData returns static interface table addresses in their encoded order.
// It reads itablinks and resolves pointer fixups for Go versions before 1.27.
// For Go 1.27 and later, it walks variable-length ITab records in the types region.
func (m moduledata) ITabLinksData() ([]uint64, error) {
	if m.fileInfo == nil || m.fileInfo.goversion == nil {
		return nil, ErrNoGoVersionFound
	}
	if m.fileInfo.WordSize != 4 && m.fileInfo.WordSize != 8 {
		return nil, errors.New("unsupported itab pointer size")
	}
	if usesGo127TypeLayout(m.fileInfo.goversion.Name) {
		if m.ITabSize == 0 {
			return nil, nil
		}
		data, err := m.itabReadRange(m.TypesAddr, m.TypesLen)
		if err != nil {
			return nil, fmt.Errorf("reading itab types region: %w", err)
		}
		parser := newTypeParser(data, m.TypesAddr, m.fileInfo, m.fh, m.resolver)
		return m.go127ITabLinks(parser)
	}
	if m.ITabLinkLen == 0 {
		return nil, nil
	}
	word := uint64(m.fileInfo.WordSize)
	if m.ITabLinkLen > ^uint64(0)/word {
		return nil, errors.New("itablinks length overflow")
	}
	data, err := m.itabReadRange(m.ITabLinkAddr, m.ITabLinkLen*word)
	if err != nil {
		return nil, fmt.Errorf("reading itablinks: %w", err)
	}
	addresses := make([]uint64, 0, len(data)/int(word))
	for offset := uint64(0); offset < uint64(len(data)); offset += word {
		value := m.itabWord(data[offset:])
		addresses = append(addresses, m.ResolvePointer(value, m.ITabLinkAddr+offset))
	}
	return addresses, nil
}

// itabReadRange checks address arithmetic and section bounds, then returns
// a view of the data without allocating a buffer of the requested size.
func (m moduledata) itabReadRange(addr, size uint64) ([]byte, error) {
	if size > ^uint64(0)-addr {
		return nil, errors.New("itab address range overflow")
	}
	base, data, err := m.fh.getSectionDataFromAddress(addr)
	if err != nil {
		return nil, err
	}
	if addr < base || addr-base > uint64(len(data)) || size > uint64(len(data))-(addr-base) {
		return nil, errors.New("itab range outside backing section")
	}
	return data[addr-base : addr-base+size], nil
}

func (m moduledata) itabWord(data []byte) uint64 {
	if m.fileInfo.WordSize == 4 {
		return uint64(m.fileInfo.ByteOrder.Uint32(data))
	}
	return m.fileInfo.ByteOrder.Uint64(data)
}

// go127ITabLinks follows runtime.addModuleItabs and internal/abi.ITab.Size.
// Sharing the caller's parser lets GetTypes include interface and concrete types
// referenced by itabs even when they are missing from the typelink descriptor prefix.
func (m moduledata) go127ITabLinks(parser *typeParser) ([]uint64, error) {
	if m.ITabOffset > m.TypesLen || m.ITabSize > m.TypesLen-m.ITabOffset ||
		m.TypesLen > uint64(len(parser.typesData)) || m.TypesLen > ^uint64(0)-m.TypesAddr {
		return nil, errors.New("itab region outside types")
	}
	if m.ITabSize == 0 {
		return nil, nil
	}
	if m.ITabOffset < m.TypeDescLen {
		return nil, errors.New("itab region overlaps type descriptors")
	}
	word := uint64(m.fileInfo.WordSize)
	funOffset := (2*word + 4 + word - 1) &^ (word - 1)
	minimum := funOffset + word
	end := m.ITabOffset + m.ITabSize
	var addresses []uint64
	for offset := m.ITabOffset; offset < end; {
		if minimum > end-offset {
			return nil, fmt.Errorf("truncated itab at %#x", m.TypesAddr+offset)
		}
		data := parser.typesData[offset:end]
		address := m.TypesAddr + offset
		interAddr := m.ResolvePointer(m.itabWord(data), address)
		typeAddr := m.ResolvePointer(m.itabWord(data[word:]), address+word)
		// Type references must stay within the preceding descriptor region.
		for _, addr := range []uint64{interAddr, typeAddr} {
			if addr < m.TypesAddr || addr-m.TypesAddr >= m.ITabOffset {
				return nil, fmt.Errorf("itab at %#x references a type outside the descriptor region: %#x", address, addr)
			}
		}
		inter, err := parser.parseType(interAddr)
		if err != nil {
			return nil, fmt.Errorf("itab at %#x interface type: %w", address, err)
		}
		if inter.Kind != reflect.Interface || len(inter.Methods) == 0 {
			return nil, fmt.Errorf("itab at %#x has an invalid interface type", address)
		}
		concrete, err := parser.parseType(typeAddr)
		if err != nil {
			return nil, fmt.Errorf("itab at %#x concrete type: %w", address, err)
		}
		if concrete.Kind == reflect.Invalid || concrete.Kind == reflect.Interface {
			return nil, fmt.Errorf("itab at %#x has an invalid concrete type", address)
		}
		methods := uint64(len(inter.Methods))
		firstFunc := m.ResolvePointer(m.itabWord(data[funOffset:]), address+funOffset)
		if firstFunc == 0 {
			methods = 1 // a cached failed assertion uses only the first slot
		}
		if methods > (end-offset-funOffset)/word {
			return nil, fmt.Errorf("itab at %#x function slots exceed region", address)
		}
		addresses = append(addresses, address)
		offset += funOffset + methods*word
	}
	return addresses, nil
}
