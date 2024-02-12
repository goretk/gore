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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Generate with version hash: {{ .Hash }}
var pkgHashMatcher = regexp.MustCompile(`// Generate with version hash: (\b[A-Fa-f0-9]{64}\b)`)

func getCurrentStdPkgHash() (string, error) {
	f, err := os.Open(stdpkgOutputFile)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	if err != nil {
		return "", err
	}

	matches := pkgHashMatcher.FindStringSubmatch(buf.String())
	if len(matches) != 2 {
		return "", nil
	}

	return matches[1], nil
}

func generateStdPkgs() {
	fmt.Println("Generating " + stdpkgOutputFile)

	collect := func(ctx context.Context, tag string, result chan []string, errChan chan error) {
		tree, _, err := githubClient.Git.GetTree(ctx, "golang", "go", tag, true)
		if err != nil {
			errChan <- fmt.Errorf("error when getting tree for tag %s: %w", tag, err)
			return
		}

		fmt.Println("Fetched std pkgs for tag: " + tag)

		if len(tree.Entries) == 100000 {
			fmt.Printf("Warning: tree %s has 100000 entries, this may be limited by api, some might be missing", tag)
		}

		var stdPkgs []string

		for _, entry := range tree.Entries {
			if *entry.Type != "tree" {
				continue
			}

			if !strings.HasPrefix(entry.GetPath(), "src/") ||
				strings.HasPrefix(entry.GetPath(), "src/cmd") ||
				strings.HasSuffix(entry.GetPath(), "_asm") ||
				strings.Contains(entry.GetPath(), "/testdata") {
				continue
			}

			stdPkgs = append(stdPkgs, strings.TrimPrefix(entry.GetPath(), "src/"))
		}
		result <- stdPkgs
	}

	f, err := os.OpenFile(goversionCsv, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		fmt.Println("Error when opening goversions.csv:", err)
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	hash, err := getFileHash(f)
	if err != nil {
		fmt.Println("Error when getting file hash:", err)
		return
	}
	currentHash, err := getCurrentStdPkgHash()
	if err != nil {
		fmt.Println("Error when getting current hash:", err)
		return
	}

	if hash == currentHash {
		fmt.Println("No need to update " + stdpkgOutputFile)
		return
	}

	knownVersions, err := getCsvStoredGoversions(f)

	stdpkgsSet := map[string]struct{}{}

	pkgsChan := make(chan []string)
	errChan := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for tag := range knownVersions {
		go collect(ctx, tag, pkgsChan, errChan)
	}

	pkgCount := 0

	for {
		select {
		case pkgs := <-pkgsChan:
			for _, pkg := range pkgs {
				stdpkgsSet[pkg] = struct{}{}
			}
			pkgCount++
			if pkgCount == len(knownVersions) {
				goto done
			}
		case err := <-errChan:
			fmt.Println("Error when collecting std pkgs:", err)
			return
		case <-ctx.Done():
			fmt.Println("Timeout when collecting std pkgs:", ctx.Err())
			return
		}
	}

done:
	pkgs := make([]string, 0, len(stdpkgsSet))
	for pkg := range stdpkgsSet {
		pkgs = append(pkgs, pkg)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i] < pkgs[j]
	})

	// Generate the code.
	buf := bytes.NewBuffer(nil)

	err = packageTemplate.Execute(buf, struct {
		Timestamp time.Time
		StdPkg    []string
		Hash      string
	}{
		Timestamp: time.Now().UTC(),
		StdPkg:    pkgs,
		Hash:      hash,
	})
	if err != nil {
		fmt.Println("Error when generating the code:", err)
		return
	}

	writeOnDemand(buf.Bytes(), stdpkgOutputFile)

	fmt.Println("Generated " + stdpkgOutputFile)
}
