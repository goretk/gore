// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var (
	goNoteNameELF  = []byte("Go\x00\x00")
	goNoteRawStart = []byte("\xff Go build ID: \"")
	goNoteRawEnd   = []byte("\"\n \xff")
)

func parseBuildIDFromElf(data []byte, byteOrder binary.ByteOrder) (string, error) {
	r := bytes.NewReader(data)
	var nameLen uint32
	var idLen uint32
	var tag uint32
	err := binary.Read(r, byteOrder, &nameLen)
	if err != nil {
		return "", fmt.Errorf("error when reading the BuildID name length: %w", err)
	}
	err = binary.Read(r, byteOrder, &idLen)
	if err != nil {
		return "", fmt.Errorf("error when reading the BuildID ID length: %w", err)
	}
	err = binary.Read(r, byteOrder, &tag)
	if err != nil {
		return "", fmt.Errorf("error when reading the BuildID tag: %w", err)
	}

	if tag != uint32(4) {
		return "", fmt.Errorf("build ID does not match expected value. 0x%x parsed", tag)
	}

	noteName := data[12 : 12+int(nameLen)]
	if !bytes.Equal(noteName, goNoteNameELF) {
		return "", fmt.Errorf("note name not as expected")
	}
	return string(data[16 : 16+int(idLen)]), nil
}

func parseBuildIDFromRaw(data []byte) (string, error) {
	idx := bytes.Index(data, goNoteRawStart)
	if idx < 0 {
		// No Build ID
		return "", nil
	}
	end := bytes.Index(data, goNoteRawEnd)
	if end < 0 {
		return "", fmt.Errorf("malformed Build ID")
	}
	return string(data[idx+len(goNoteRawStart) : end]), nil
}
