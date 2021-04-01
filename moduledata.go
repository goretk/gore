// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

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
