// This file is part of GoRE.
//
// Copyright (C) 2019-2022 GoRE Authors
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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuledata(t *testing.T) {
	r := require.New(t)

	cases := []struct {
		file                    string
		text, textLen           uint64
		noptrdata, noptrdataLen uint64
		data, dataLen           uint64
		bss, bssLen             uint64
		noptrbss, noptrbssLen   uint64
		gofunc                  uint64
	}{
		{"gold-linux-amd64-1.20.0", 0x00401000, 0x80f84, 0x515180, 0x105c0, 0x525740, 0x78f0, 0x52d040, 0x2dd60, 0x55ada0, 0x39d0, 0x4a1e88},
		{"gold-linux-amd64-1.16.0", 0x00401000, 0x98277, 0x00538020, 0xe2c4, 0x00546300, 0x7790, 0x0054daa0, 0x2d750, 0x0057b200, 0x5310, 0},
		{"gold-linux-amd64-1.8.0", 0x00401000, 0x7c7d0, 0x004f9000, 0x24c8, 0x004fb4e0, 0x1d10, 0x004fd200, 0x1a908, 0x00517b20, 0x46a0, 0},
		{"gold-linux-amd64-1.7.0", 0x00401000, 0x7d2a0, 0x004f8000, 0x2048, 0x004fa060, 0x1d70, 0x004fbde0, 0x1a910, 0x00516700, 0x4e80, 0},
		{"gold-linux-amd64-1.5.0", 0x00401000, 0xb15f0, 0x00590000, 0x1bc8, 0x00591be0, 0x2550, 0x00594140, 0x23bd8, 0x005b7d20, 0x4e40, 0},
		{"gold-linux-386-1.16.0", 0x08049000, 0x846d9, 0x0815a020, 0xdda4, 0x08167de0, 0x3c28, 0x0816ba20, 0x11cfc, 0x0817d720, 0x45e0, 0},
	}

	for _, test := range cases {
		t.Run("moduledata-"+test.file, func(t *testing.T) {
			f, err := Open(filepath.Join("testdata", "gold", test.file))
			r.NoError(err)
			defer f.Close()

			md, err := f.Moduledata()
			r.NoError(err)

			mdSec := md.Text()
			r.Equal(test.text, mdSec.Address)
			r.Equal(test.textLen, mdSec.Length)

			mdSec = md.NoPtrData()
			r.Equal(test.noptrdata, mdSec.Address)
			r.Equal(test.noptrdataLen, mdSec.Length)

			mdSec = md.Data()
			r.Equal(test.data, mdSec.Address)
			r.Equal(test.dataLen, mdSec.Length)

			mdSec = md.Bss()
			r.Equal(test.bss, mdSec.Address)
			r.Equal(test.bssLen, mdSec.Length)

			mdSec = md.NoPtrBss()
			r.Equal(test.noptrbss, mdSec.Address)
			r.Equal(test.noptrbssLen, mdSec.Length)

			mdSec = md.PCLNTab()
			r.NotEqual(0, mdSec.Address)
			r.NotEqual(0, mdSec.Length)

			mdSec = md.ITabLinks()
			r.NotEqual(0, mdSec.Address)
			r.NotEqual(0, mdSec.Length)

			mdSec = md.FuncTab()
			r.NotEqual(0, mdSec.Address)
			r.NotEqual(0, mdSec.Length)

			r.Equal(test.gofunc, md.GoFuncValue())
		})
	}
}
