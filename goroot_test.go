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
			r.NoError(err)
			r.Equal(expectGoRoot, goroot)
		})
	}
}
