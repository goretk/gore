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
