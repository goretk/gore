// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
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
	"encoding/binary"
)

// keep sync with debug/gosym/pclntab.go
const (
	gopclntab12magic  uint32 = 0xfffffffb
	gopclntab116magic uint32 = 0xfffffffa
	gopclntab118magic uint32 = 0xfffffff0
	gopclntab120magic uint32 = 0xfffffff1
)

// searchSectionForTab looks for the PCLN table within the section.
func searchSectionForTab(secData []byte, order binary.ByteOrder) ([]byte, error) {
	// First check for the current magic used. If this fails, it could be
	// an older version. So check for the old header.
MagicLoop:
	for _, magic := range [][]byte{pclntab120magic, pclntab118magic, pclntab116magic, pclntab12magic} {
		off := bytes.LastIndex(secData, magic)
	for _, magic := range []uint32{gopclntab120magic, gopclntab118magic, gopclntab116magic, gopclntab12magic} {
		bMagic := make([]byte, 6) // 4 bytes for the magic, 2 bytes for padding.
		order.PutUint32(bMagic, magic)

		off := bytes.LastIndex(secData, bMagic)
		if off == -1 {
			continue // Try other magic.
		}
		for off != -1 {
			if off != 0 {
				buf := secData[off:]
				if len(buf) < 16 || buf[4] != 0 || buf[5] != 0 ||
					(buf[6] != 1 && buf[6] != 2 && buf[6] != 4) || // pc quantum
					(buf[7] != 4 && buf[7] != 8) { // pointer size
					// Header doesn't match.
					if off-1 <= 0 {
						continue MagicLoop
					}
					off = bytes.LastIndex(secData[:off-1], bMagic)
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
