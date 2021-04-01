// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"debug/pe"
)

// pclntabmagic is the magic bytes used for binaries compiled with Go
// prior to 1.16
var pclntabmagic = []byte{0xfb, 0xff, 0xff, 0xff}

// pclntab116magic is the magic bytes used for binaries compiled with
// Go 1.16 and onwards.
var pclntab116magic = []byte{0xfa, 0xff, 0xff, 0xff}

// searchFileForPCLNTab will search the .rdata and .text section for the
// PCLN table. Note!! The address returned by this function needs to be
// adjusted by adding the image base address!!!
func searchFileForPCLNTab(f *pe.File) (uint32, []byte, error) {
	for _, v := range []string{".rdata", ".text"} {
		sec := f.Section(v)
		if sec == nil {
			continue
		}
		secData, err := sec.Data()
		if err != nil {
			continue
		}
		tab, err := searchSectionForTab(secData)
		if err == ErrNoPCLNTab {
			continue
		}
		// TODO: Switch to returning a uint64 instead.
		addr := sec.VirtualAddress + uint32(len(secData)-len(tab))
		return addr, tab, err
	}
	return 0, []byte{}, ErrNoPCLNTab
}

// searchSectionForTab looks for the PCLN table within the section.
func searchSectionForTab(secData []byte) ([]byte, error) {
	// First check for the current magic used. If this fails, it could be
	// an older version. So check for the old header.
	for _, magic := range [][]byte{pclntab116magic, pclntabmagic} {
		off := bytes.LastIndex(secData, magic)
		if off == -1 {
			return nil, ErrNoPCLNTab
		}
		for off != -1 {
			if off != 0 {
				buf := secData[off:]
				if len(buf) < 16 || buf[4] != 0 || buf[5] != 0 ||
					(buf[6] != 1 && buf[6] != 2 && buf[6] != 4) || // pc quantum
					(buf[7] != 4 && buf[7] != 8) { // pointer size
					// Header doesn't match.
					if off-1 <= 0 {
						return nil, ErrNoPCLNTab
					}
					off = bytes.LastIndex(secData[:off-1], pclntabmagic)
					continue
				}
				// Header match
				return secData[off:], nil
			}
			break
		}
	}
	return nil, ErrNoPCLNTab
}
