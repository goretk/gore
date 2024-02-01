// This file is part of GoRE.
//
// Copyright (C) 2019-2024 GoRE Authors
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
	"bufio"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// diffCode returns false if a and b have different other than the date.
func diffCode(a, b string) bool {
	if a == b {
		return false
	}

	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	// ignore the license and the date
	aLines = aLines[21:]
	bLines = bLines[21:]

	if len(aLines) != len(bLines) {
		return true
	}

	for i := 0; i < len(aLines); i++ {
		if aLines[i] != bLines[i] {
			return true
		}
	}

	return false
}

func writeOnDemand(new []byte, target string) {
	old, err := os.ReadFile(target)
	if err != nil {
		fmt.Println("Error when reading the old file:", target, err)
		return
	}

	old, _ = format.Source(old)
	new, _ = format.Source(new)

	// Compare the old and the new.
	if !diffCode(string(old), string(new)) {
		fmt.Println(target + " no changes.")
		return
	}

	fmt.Println(target + " changes detected.")

	// Write the new file.
	err = os.WriteFile(target, new, 0664)
	if err != nil {
		fmt.Println("Error when writing the new file:", err)
		return
	}
}

func getSourceDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return filepath.Join(filepath.Dir(filename), "..")
}

func getCsvStoredGoversions(f *os.File) (map[string]*goversion, error) {
	vers := make(map[string]*goversion)
	r := bufio.NewScanner(f)
	// Read header
	if !r.Scan() {
		return nil, errors.New("empty file")
	}
	r.Text()

	for r.Scan() {
		row := r.Text()
		if row == "" {
			continue
		}
		data := strings.Split(row, ",")
		if data[0] == "" {
			// No version
			continue
		}
		version := strings.TrimSpace(data[0])
		sha := strings.TrimSpace(data[1])
		date := strings.TrimSpace(data[2])
		vers[version] = &goversion{Name: version, Sha: sha, Date: date}
	}
	_, err := f.Seek(0, 0)
	return vers, err
}
