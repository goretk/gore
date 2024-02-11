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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildInfo(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}

	for _, test := range goldFiles {
		t.Run("extracting build info for "+test, func(t *testing.T) {
			r := require.New(t)

			fp, err := getTestResourcePath("gold/" + test)
			r.NoError(err, "Failed to get path to resource")

			if _, err = os.Stat(fp); os.IsNotExist(err) {
				// Skip this file because it doesn't exist
				t.Skipf("[SKIPPING TEST] golden fille %s does not exist\n", test)
			}

			f, err := Open(fp)
			r.NoError(err)

			ver, err := f.GetCompilerVersion()
			r.NoError(err)

			if GoVersionCompare(ver.Name, "go1.13") == -1 {
				// No build info available for these builds.
				r.Nil(f.BuildInfo)
			} else {
				// Version with build info.
				r.NotNil(f.BuildInfo)
				r.NotNil(f.BuildInfo.Compiler)
				if GoVersionCompare(ver.Name, "go1.16") >= 0 {
					// The mod info is not always available in Go versions earlier than 1.16.

					//if GoVersionCompare(ver.Name, "go1.19beta1") >= 0 {
					//	r.Equal("github.com/goretk/gore/gold", f.BuildInfo.ModInfo.Path)
					//} else {
					//	r.Equal("command-line-arguments", f.BuildInfo.ModInfo.Path)
					//}
					switch {
					case GoVersionCompare(ver.Name, "go1.19beta1") >= 0:
						r.Equal("github.com/goretk/gore/gold", f.BuildInfo.ModInfo.Path)
					case GoVersionCompare(ver.Name, "go1.16.0") >= 0 && strings.Contains(test, "darwin-arm64"):
						r.Equal("github.com/goretk/gore/gold", f.BuildInfo.ModInfo.Path)
					default:
						r.Equal("command-line-arguments", f.BuildInfo.ModInfo.Path)
					}

					//if GoVersionCompare(ver.Name, "go1.19beta1") >= 0 {
					//	r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
					//} else if GoVersionCompare(ver.Name, "go1.18beta1") >= 0 {
					//	r.Equal("", f.BuildInfo.ModInfo.Main.Version)
					//} else {
					//	r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
					//}
					switch {
					case GoVersionCompare(ver.Name, "go1.19beta1") >= 0:
						r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
					case GoVersionCompare(ver.Name, "go1.18beta1") >= 0 && strings.Contains(test, "darwin-arm64"):
						r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
					case GoVersionCompare(ver.Name, "go1.18beta1") >= 0:
						r.Equal("", f.BuildInfo.ModInfo.Main.Version)
					default:
						r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
					}
				}
			}
		})
	}
}
