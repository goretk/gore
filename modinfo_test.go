// Copyright 2021 The GoRE Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"os"
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
					r.Equal("command-line-arguments", f.BuildInfo.ModInfo.Main.Path)
					r.Equal("(devel)", f.BuildInfo.ModInfo.Main.Version)
				}
			}
		})
	}
}
