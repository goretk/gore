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
	"encoding/csv"
	"fmt"
	"github.com/google/go-github/v58/github"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func generateGoVersions() {
	fmt.Println("Generating " + goversionOutputFile + "&" + goversionCsv)

	ctx := context.Background()

	opts := &github.ListOptions{PerPage: 100}
	var allTags []*github.RepositoryTag
	for {
		tags, resp, err := githubClient.Repositories.ListTags(ctx, "golang", "go", opts)
		if err != nil {
			fmt.Println(err)
			return
		}
		allTags = append(allTags, tags...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
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

	for _, tag := range allTags {
		if strings.HasPrefix(tag.GetName(), "weekly") || strings.HasPrefix(tag.GetName(), "release") {
			continue
		}
		if _, known := knownVersions[tag.GetName()]; known {
			continue
		}

		commit, _, err := githubClient.Repositories.GetCommit(ctx, "golang", "go", tag.GetCommit().GetSHA(), nil)
		if err != nil {
			fmt.Println("Error when getting commit info:", err)
			return
		}

		fmt.Println("New tag found:", tag.GetName())
		knownVersions[tag.GetName()] = &goversion{Name: tag.GetName(), Sha: commit.GetSHA(), Date: commit.GetCommit().GetCommitter().GetDate().Format(time.RFC3339)}
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

	// Generate the code.

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

	fmt.Println("Generated " + goversionOutputFile + "&" + goversionCsv)
}
