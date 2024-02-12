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
	"encoding/csv"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func generateGoVersions() {
	fmt.Println("Generating " + goversionOutputFile + " & " + goversionCsv)

	tags, err := goRepo.Tags()
	if err != nil {
		fmt.Println("Error when getting tags:", err)
		return
	}

	f, err := os.OpenFile(goversionCsv, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		fmt.Println("Error when opening goversions.csv:", err)
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	knownVersions, err := getCsvStoredGoversions(f)
	if err != nil {
		fmt.Println("Error when getting stored go versions:", err)
		return
	}

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		name := tag.Name().Short()
		if strings.HasPrefix(name, "weekly") || strings.HasPrefix(name, "release") {
			return nil
		}

		if _, known := knownVersions[name]; known {
			return nil
		}

		commit, err := goRepo.CommitObject(tag.Hash())
		if err != nil {
			return err
		}

		fmt.Println("New tag found:", name)

		knownVersions[name] = &goversion{Name: name, Sha: commit.Hash.String(), Date: commit.Committer.When.Format(time.RFC3339)}

		return nil
	})
	if err != nil {
		fmt.Println("Error when getting tags:", err)
		return
	}

	sortedVersion := make([]*goversion, 0, len(knownVersions))
	for _, ver := range knownVersions {
		sortedVersion = append(sortedVersion, ver)
	}

	sort.Slice(sortedVersion, func(i, j int) bool {
		time1, err := time.Parse(time.RFC3339, sortedVersion[i].Date)
		if err != nil {
			fmt.Println("Error when parsing time:", err)
			return false
		}
		time2, err := time.Parse(time.RFC3339, sortedVersion[j].Date)
		if err != nil {
			fmt.Println("Error when parsing time:", err)
			return false
		}
		return time1.Before(time2)
	})

	// Generate the csv
	err = f.Truncate(0)
	if err != nil {
		fmt.Println("Error when truncating the file:", err)
		return
	}
	_, _ = f.Seek(0, io.SeekStart)

	cw := csv.NewWriter(f)
	for _, ver := range sortedVersion {
		_ = cw.Write([]string{ver.Name, ver.Sha, ver.Date})
	}
	cw.Flush()

	// Generate the code
	buf := bytes.NewBuffer(nil)
	err = goversionTemplate.Execute(buf, struct {
		Timestamp  time.Time
		GoVersions []*goversion
	}{
		Timestamp:  time.Now().UTC(),
		GoVersions: sortedVersion,
	})
	if err != nil {
		fmt.Println("Error when generating the code:", err)
		return
	}

	writeOnDemand(buf.Bytes(), goversionOutputFile)

	fmt.Println("Generated " + goversionOutputFile + " & " + goversionCsv)
}
