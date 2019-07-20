// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

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
