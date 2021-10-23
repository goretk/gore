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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGo116PCLNTab(t *testing.T) {
	r := require.New(t)

	files := []string{
		"gold-linux-amd64-1.16.0",
		"gold-darwin-amd64-1.16.0",
		"gold-windows-amd64-1.16.0",
	}

	for _, gold := range files {
		t.Run(gold, func(t *testing.T) {
			win := filepath.Join("testdata", "gold", gold)

			f, err := Open(win)
			r.NoError(err)

			tab, err := f.PCLNTab()
			r.NoError(err)
			r.NotNil(tab)
		})
	}

}
