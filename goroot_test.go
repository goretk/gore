// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestExtractGoRoot(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}
	var expectGoRoot string = "/usr/local/go"
	for _, test := range goldFiles {
		t.Run("get goroot form "+test, func(t *testing.T) {
			r := require.New(t)
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
			// Windows version 1.5 to 1.9 did not find a goroot string that can be searched, so we ruled it out
			switch test {
			case "windows-386-1.5.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-386-1.6.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-386-1.7.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-386-1.7beta1":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-386-1.8.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-386-1.9.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.5.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.6.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.7.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.7beta1":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.8.0":
				r.Equal(ErrNoGoRootFound, err)
			case "windows-amd64-1.9.0":
				r.Equal(ErrNoGoRootFound, err)
			default:
				r.NoError(err)
				r.Equal(expectGoRoot, goroot)
			}
		})
	}
}
