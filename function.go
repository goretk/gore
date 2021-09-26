// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"debug/gosym"
	"fmt"
	"sort"
	"strings"
)

// Function is a representation of a Go function.
type Function struct {
	// Name is the extracted function name.
	Name string `json:"name"`
	// Offset is the starting location for the subroutine in the binary.
	Offset uint64 `json:"offset"`
	// End is the end location for the subroutine in the binary.
	End uint64 `json:"end"`
	// PackageName is the name of the Go package the function belongs to.
	PackageName string `json:"packageName"`
}

// String returns a string representation of the function.
func (f *Function) String() string {
	return f.Name
}

// Method is a representation of a Go method.
type Method struct {
	// Receiver is the name of the method receiver.
	Receiver string `json:"receiver"`
	*Function
}

// String returns a string summary of the function.
func (m *Method) String() string {
	return fmt.Sprintf("%s%s", m.Receiver, m.Name)
}

// FileEntry is a representation of an entry in a source code file. This can for example be
// a function or a method.
type FileEntry struct {
	// Name of the function or method.
	Name string
	// Start is the source line where the code starts.
	Start int
	// End is the source line where the code ends.
	End int
}

// String returns a string representation of the entry.
func (f FileEntry) String() string {
	return fmt.Sprintf("%s Lines: %d to %d (%d)", f.Name, f.Start, f.End, f.End-f.Start)
}

// SourceFile is a representation of a source code file.
type SourceFile struct {
	// Name of the file.
	Name string
	// Prefix that should be added to each line.
	Prefix string
	// Postfix that should be added to each line.
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
		return s.entries[i].Start < s.entries[j].Start
	})

	numlines := len(s.entries) + 1
	lines := make([]string, numlines)
	lines[0] = fmt.Sprintf("%sFile: %s%s", s.Prefix, s.Name, s.Postfix)

	// Entry lines
	for i, e := range s.entries {
		lines[i+1] = fmt.Sprintf("\t%s%s%s", s.Prefix, e, s.Postfix)
	}
	return strings.Join(lines, "\n")
}

// findSourceLines walks from the entry of the function to the end and looks for the
// final source code line number. This function is pretty expensive to execute.
func findSourceLines(entry, end uint64, tab *gosym.Table) (int, int) {
	// We don't need the Func returned since we are operating within the same function.
	file, srcStart, _ := tab.PCToLine(entry)

	// We walk from entry to end and check the source code line number. If it's greater
	// then the current value, we set it as the new value. If the file is different, we
	// have entered an inlined function. In this case we skip it. There is a possibility
	// that we enter an inlined function that's defined in the same file. There is no way
	// for us to tell this is the case.
	srcEnd := srcStart

	// We take a shortcut and only check every 4 bytes. This isn't perfect, but it speeds
	// up the processes.
	for i := entry; i <= end; i = i + 4 {
		f, l, _ := tab.PCToLine(i)

		// If this line is a different file, it's an inlined function so just continue.
		if f != file {
			continue
		}

		// If the current line is less than the starting source line, we have entered
		// an inline function defined before this function.
		if l < srcStart {
			continue
		}

		// If the current line is greater, we assume it being closer to the end of the
		// function definition. So we take it as the current srcEnd value.
		if l > srcEnd {
			srcEnd = l
		}
	}

	return srcStart, srcEnd
}
