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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractGoRoot(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}

	const expectGoRoot = "/usr/local/go"

	for _, test := range goldFiles {
		t.Run("get goroot form "+test, func(t *testing.T) {
			r := require.New(t)

			// TODO: Remove this check when arm support has been added.
			if strings.Contains(test, "arm64") {
				t.Skip("ARM currently not supported")
			}

			fp, err := getTestResourcePath("gold/" + test)
			r.NoError(err, "Failed to get path to resource")
			if _, err = os.Stat(fp); os.IsNotExist(err) {
				// Skip this file because it doesn't exist
				// t.Skip will cause the parent test to be skipped.
				fmt.Printf("[SKIPPING TEST] golden fille %s does not exist\n", test)
				return
			}
			r.NoError(err)
			f, err := Open(fp)
			r.NoError(err)
			defer f.Close()
			goroot, err := findGoRootPath(f)
			r.NoError(err)
			r.Equal(expectGoRoot, goroot)
		})
	}
}
