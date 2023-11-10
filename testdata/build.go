// This file is part of GoRE.
//
// # Copyright (C) 2023 GoRE Authors
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
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// If no user cache dir, use global tmp instead.
		cacheDir = ""
	}
	buildDir, err := os.MkdirTemp(cacheDir, "gold-builds-*")
	if err != nil {
		fmt.Println("Failed to create a build folder:", err)
		return
	}
	defer os.RemoveAll(buildDir)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("No current working director:", err)
		return
	}
	goldFolder := filepath.Join(cwd, "gold")

	err = os.WriteFile(filepath.Join(buildDir, "target.go"), []byte(gofile), 0644)
	if err != nil {
		fmt.Printf("Error when writing template file to build folder: %s.\n", err)
		return
	}

	// Add go.mod file
	err = os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte(gomodstub), 0644)
	if err != nil {
		fmt.Printf("Error when writing go.mod file to build folder: %s.\n", err)
		return
	}

	// Enumerate missing golden binaries.
	var missing []goversionEntry
	for _, v := range spec {
		ver := v.version
		for _, o := range v.variants {
			oper := o.os
			for _, arch := range o.arch {
				f := fmt.Sprintf("gold-%s-%s-%s", oper, arch, ver)
				if _, err = os.Stat(filepath.Join(goldFolder, f)); errors.Is(err, os.ErrNotExist) {
					missing = append(missing, goversionEntry{ver: ver, arch: arch, os: oper})
				}
			}
		}
	}
	if len(missing) == 0 {
		fmt.Println("No missing files.")
		return
	}
	fmt.Println("Missing files:")
	for _, f := range missing {
		fmt.Println(f)
	}

	// Try to build using Docker container.
	for _, v := range missing {
		cmd := exec.Command(
			"docker",
			"run",
			"--rm",
			"-e", "GOOS="+string(v.os),
			"-e", "GOARCH="+string(v.arch),
			"-v", buildDir+":/build",
			"-w", "/build/",
			"golang:"+v.ver,
			"go", "build", "-ldflags", "-s -w",
			"-o", "/build/"+v.String(),
		)
		fmt.Println("Try to build:", v)
		stderr := bytes.Buffer{}
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println("Execution failed:", err)
			fmt.Println("ERR:", stderr.String())
		} else {
			fmt.Println("Successfuly built:", v)
		}

		// Move the file to the golden folder.
		err = os.Rename(filepath.Join(buildDir, v.String()), filepath.Join(goldFolder, v.String()))
		if err != nil {
			fmt.Printf("Error when moving %s to golden folder: %s.\n", v.String(), err)
		}
	}
}

type goos string
type goarch string

const (
	linux   goos   = "linux"
	darwin  goos   = "darwin"
	windows goos   = "windows"
	x86     goarch = "386"
	amd64   goarch = "amd64"
	arm64   goarch = "arm64"
)

type osarchTuple struct {
	os   goos
	arch []goarch
}

type goversionEntry struct {
	ver  string
	arch goarch
	os   goos
}

func (e goversionEntry) String() string {
	return fmt.Sprintf("gold-%s-%s-%s", e.os, e.arch, e.ver)
}

var spec = []struct {
	version  string
	variants []osarchTuple
}{
	{"1.5.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.6.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.7beta1", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.7.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.8.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.9.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.10.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.11.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.12.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.13.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.14.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{x86, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.15.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.16.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.17.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.18.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.19.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.20.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
	{"1.21.0", []osarchTuple{{linux, []goarch{x86, amd64}}, {darwin, []goarch{arm64, amd64}}, {windows, []goarch{x86, amd64}}}},
}

const gofile = `package main

import "fmt"

type myComplexStruct struct {
	MyString string "json:\"String\""
	person   *simpleStruct
	myArray  [2]int
	mySlice  []uint
	myChan   chan struct{}
	myMap    map[string]int
	myFunc   func(string, int) uint
	embeddedType
}

type simpleStruct struct {
	name string
	age  int
}

func (s *simpleStruct) String() string {
	return fmt.Sprintf("Name: %s | Age: %d", s.name, s.age)
}

type embeddedType struct {
	val int64
}

func main() {
	myPerson := &simpleStruct{name: "Test string", age: 42}
	complexStruct := &myComplexStruct{MyString: "A string", person: myPerson}
	fmt.Printf("Person: %v and a struct %v\n", myPerson, complexStruct)
}
`
const gomodstub = `module github.com/goretk/gore/gold

go 1.14
`
