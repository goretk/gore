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
	"errors"
	"fmt"
	"io"
)

type moduledata struct {
	// typesAddr address
	typesAddr uint64
	// typelinksAddr is the address to the typelink
	typelinkAddr uint64
	// typelinksLen is the length of the typelink
	typelinkLen uint64
}

func findModuledata(f fileHandler) ([]byte, error) {
	_, secData, err := f.getSectionData(f.moduledataSection())
	if err != nil {
		return nil, err
	}
	tabAddr, _, err := f.getPCLNTABData()
	if err != nil {
		return nil, err
	}

	// Search for moduledata
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, &tabAddr)
	if err != nil {
		return nil, err
	}
	off := bytes.Index(secData, buf.Bytes()[:intSize32])
	if off == -1 {
		return nil, errors.New("could not find moduledata")
	}
	// TODO: Verify that hit is correct.

	return secData[off : off+0x300], nil
}

func parseModuledata(fileInfo *FileInfo, f fileHandler) (*moduledata, error) {
	data, err := findModuledata(f)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(data)
	md := new(moduledata)

	// Parse types
	if GoVersionCompare("go1.16beta1", fileInfo.goversion.Name) <= 0 {
		r.Seek(int64(35*fileInfo.WordSize), io.SeekStart)
	} else {
		r.Seek(int64(25*fileInfo.WordSize), io.SeekStart)
	}

	typeAddr, err := readUIntTo64(r, fileInfo.ByteOrder, fileInfo.WordSize == intSize32)
	if err != nil {
		return nil, err
	}
	md.typesAddr = typeAddr

	if GoVersionCompare("go1.16beta1", fileInfo.goversion.Name) <= 0 {
		r.Seek(int64(40*fileInfo.WordSize), io.SeekStart)
	} else if GoVersionCompare("go1.8beta1", fileInfo.goversion.Name) <= 0 {
		r.Seek(int64(30*fileInfo.WordSize), io.SeekStart)
	} else if GoVersionCompare("go1.7beta1", fileInfo.goversion.Name) <= 0 {
		r.Seek(int64(27*fileInfo.WordSize), io.SeekStart)
	} else {
		// Legacy
		r.Seek(int64(25*fileInfo.WordSize), io.SeekStart)
	}

	typelinkAddr, err := readUIntTo64(r, fileInfo.ByteOrder, fileInfo.WordSize == intSize32)
	if err != nil {
		return nil, fmt.Errorf("failed to read typelink addres: %w", err)
	}
	md.typelinkAddr = typelinkAddr
	typelinkLen, err := readUIntTo64(r, fileInfo.ByteOrder, fileInfo.WordSize == intSize32)
	if err != nil {
		return nil, fmt.Errorf("failed to read typelink length: %w", err)
	}
	md.typelinkLen = typelinkLen

	return md, nil
}

func readUIntTo64(r io.Reader, byteOrder binary.ByteOrder, is32bit bool) (uint64, error) {
	if is32bit {
		var addr uint32
		err := binary.Read(r, byteOrder, &addr)
		return uint64(addr), err
	}
	var addr uint64
	err := binary.Read(r, byteOrder, &addr)
	return addr, err
}
