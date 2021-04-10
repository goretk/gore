// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringer(t *testing.T) {
	assert := assert.New(t)

	// Setup
	mainFunc := &Function{Name: "main", SrcLineStart: 3, SrcLineEnd: 6, SrcLineLength: 4}
	mainFuncStr := "main Lines: 3 to 6 (4)"
	setupFunc := &Function{Name: "setup", SrcLineStart: 8, SrcLineEnd: 11, SrcLineLength: 4}
	setupFuncStr := "setup Lines: 8 to 11 (4)"

	tests := []struct {
		name     string
		obj      fmt.Stringer
		expected string
	}{
		{"main", mainFunc, mainFuncStr},
		{"method", &Method{Receiver: "(myStruct)", Function: &Function{Name: "myMethod", SrcLineStart: 3, SrcLineEnd: 6, SrcLineLength: 4}}, "(myStruct)myMethod Lines: 3 to 6 (4)"},
		{"SourceFile", &SourceFile{Name: "main.go", entries: []FileEntry{mainFunc, setupFunc}}, fmt.Sprintf("File: %s\n\t%s\n\t%s", "main.go", mainFuncStr, setupFuncStr)},
		{"SourceFile_unsorted", &SourceFile{Name: "main.go", entries: []FileEntry{setupFunc, mainFunc}}, fmt.Sprintf("File: %s\n\t%s\n\t%s", "main.go", mainFuncStr, setupFuncStr)},
		{"SourceFile_prefix", &SourceFile{Name: "main.go", entries: []FileEntry{setupFunc, mainFunc}, Prefix: "\t"}, fmt.Sprintf("\tFile: %s\n\t\t%s\n\t\t%s", "main.go", mainFuncStr, setupFuncStr)},
		{"SourceFile_postfix", &SourceFile{Name: "main.go", entries: []FileEntry{setupFunc, mainFunc}, Postfix: "\t"}, fmt.Sprintf("File: %s\t\n\t%s\t\n\t%s\t", "main.go", mainFuncStr, setupFuncStr)},
	}

	for _, test := range tests {
		t.Run("stringer_"+test.name, func(t *testing.T) {
			assert.Equal(test.expected, test.obj.String())
		})
	}

}
