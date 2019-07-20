// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"sort"
	"strings"
)

// Function is a representation of a Go function.
type Function struct {
	// Name is the extracted function name.
	Name string `json:"name"`
	// SrcLineLength is the number of source code lines for the function.
	SrcLineLength int `json:"srcLength"`
	// SrcLineStart is the starting source code line number for the function.
	SrcLineStart int `json:"srcStart"`
	// SrcLineEnd is the ending source code line number for the function.
	SrcLineEnd int `json:"srcEnd"`
	// Offset is the starting location for the subroutine in the binary.
	Offset uint64 `json:"offset"`
	// End is the end location for the subroutine in the binary.
	End uint64 `json:"end"`
	// Filename is name of the name of the source code file for the function.
	Filename string `json:"filename"`
	// PackageName is the name of the Go package the function belongs to.
	PackageName string `json:"packageName"`
}

// String returns a string summary of the function.
func (f *Function) String() string {
	return fmt.Sprintf("%s Lines: %d to %d (%d)", f.Name, f.SrcLineStart, f.SrcLineEnd, f.SrcLineLength)
}

// LineStart is the first source code line for the function.
func (f *Function) LineStart() int {
	return f.SrcLineStart
}

// Method is a representation of a Go method.
type Method struct {
	// Receiver is the name of the method receiver.
	Receiver string `json:"receiver"`
	*Function
}

// String returns a string summary of the function.
func (m *Method) String() string {
	return fmt.Sprintf("%s%s Lines: %d to %d (%d)", m.Receiver, m.Name, m.SrcLineStart, m.SrcLineEnd, m.SrcLineLength)
}

// FileEntry is a representation of an entry in a source code file. This can for example be
// a function or a method.
type FileEntry interface {
	fmt.Stringer
	LineStart() int
}

// SourceFile is a representation of a source code file.
type SourceFile struct {
	Name    string
	Prefix  string
	Postfix string
	entries []FileEntry
}

// String produces a string representation of a source code file.
// The multi-line string has this format:
//		File: simple.go
//			main Lines: 5 to 8 (3)
//			setup Lines: 9 to 11 (2)
// The prefix and postfix string is added to each line.
func (s *SourceFile) String() string {
	sort.Slice(s.entries, func(i, j int) bool {
		return s.entries[i].LineStart() < s.entries[j].LineStart()
	})
	numlines := len(s.entries) + 1
	lines := make([]string, numlines, numlines)
	lines[0] = fmt.Sprintf("%sFile: %s%s", s.Prefix, s.Name, s.Postfix)

	// Entry lines
	for i, e := range s.entries {
		lines[i+1] = fmt.Sprintf("\t%s%s%s", s.Prefix, e, s.Postfix)
	}
	return strings.Join(lines, "\n")
}
