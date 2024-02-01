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
	"github.com/google/go-github/v58/github"
	"os"
	"strings"
	"time"
)

func generateGoVersions() {
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

	// Get mode commit info for new tags

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

	_, err = fmt.Fprintln(f, "version,sha,date")
	if err != nil {
		fmt.Println("Error when writing csv header:", err)
		return
	}

	for _, tag := range allTags {
		if strings.HasPrefix(tag.GetName(), "weekly") || strings.HasPrefix(tag.GetName(), "release") {
			continue
		}
		if v, known := knownVersions[tag.GetName()]; known {
			_, _ = fmt.Fprintf(f, "%s,%s,%s\n", v.Name, v.Sha, v.Date)
			continue
		}

		commit, _, err := githubClient.Repositories.GetCommit(ctx, "golang", "go", tag.GetCommit().GetSHA(), nil)
		if err != nil {
			fmt.Println("Error when getting commit info:", err)
			return
		}
		_, _ = fmt.Fprintf(f, "%s,%s,%s\n", tag.GetName(), commit.GetSHA(), commit.GetCommitter().GetCreatedAt().String())
		fmt.Println("New tag found:", tag.Name)
		knownVersions[tag.GetName()] = &goversion{Name: tag.GetName(), Sha: commit.GetSHA(), Date: commit.GetCommitter().GetCreatedAt().String()}
	}

	// Generate the code.
	buf := bytes.NewBuffer(nil)

	err = goversionTemplate.Execute(buf, struct {
		Timestamp  time.Time
		GoVersions map[string]*goversion
	}{
		Timestamp:  time.Now().UTC(),
		GoVersions: knownVersions,
	})
	if err != nil {
		fmt.Println("Error when generating the code:", err)
		return
	}

	writeOnDemand(buf.Bytes(), goversionOutputFile)
}
