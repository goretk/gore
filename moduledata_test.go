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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuledata(t *testing.T) {
	t.SkipNow() // Skip until new resource is found.
	require := require.New(t)
	assert := assert.New(t)
	fp, err := getTestResourcePath("elf64")
	require.NoError(err, "Failed to get path to resource")
	f, err := Open(fp)
	require.NoError(err, "Failed to get path elf file")

	ver, _ := f.GetCompilerVersion()
	f.FileInfo.goversion = ver
	md, err := parseModuledata(f.FileInfo, f.fh)

	assert.NoError(err)
	// rdata := uint64(uint32(f.file.Section(".rodata").Addr))
	// assert.Equal(rdata, md.typesAddr, "Incorrect types address")
	assert.Equal(uint64(0x4c9f20), md.typelinkAddr, "Wrong typelink address")
	assert.Equal(uint64(0x2d7), md.typelinkLen, "Wrong typelink length")
}
