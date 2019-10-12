// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBuildIDElf(t *testing.T) {
	assert := assert.New(t)
	expectedID := "DrtsigZmOidE-wfbFVNF/io-X8KB-ByimyyODdYUe/Z7tIlu8GbOwt0Jup-Hji/fofocVx5sk8UpaKMTx0a"
	tag := uint32(4)
	gonote := []byte("Go\x00\x00")
	nameSize := uint32(4)

	buf := &bytes.Buffer{}

	binary.Write(buf, binary.LittleEndian, nameSize)
	binary.Write(buf, binary.LittleEndian, uint32(len(expectedID)))
	binary.Write(buf, binary.LittleEndian, tag)
	buf.Write(gonote)
	buf.Write([]byte(expectedID))

	actual, err := parseBuildIDFromElf(buf.Bytes(), binary.LittleEndian)
	assert.NoError(err, "Parsing the note should not fail.")
	assert.Equal(expectedID, actual, "Extracted ID does not match.")
}

func TestParseBuildIDRaw(t *testing.T) {
	assert := assert.New(t)
	expectedID := "DrtsigZmOidE-wfbFVNF/io-X8KB-ByimyyODdYUe/Z7tIlu8GbOwt0Jup-Hji/fofocVx5sk8UpaKMTx0a"

	buf := &bytes.Buffer{}
	buf.Write(goNoteRawStart)
	buf.Write([]byte(expectedID))
	buf.Write(goNoteRawEnd)
	buf.Write([]byte{0xcc, 0xcc, 0xcc, 0xcc})

	actual, err := parseBuildIDFromRaw(buf.Bytes())
	assert.NoError(err, "Parsing the note should not fail.")
	assert.Equal(expectedID, actual, "Extracted ID does not match.")
}
