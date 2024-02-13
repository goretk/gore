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
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/go-git/go-git/v5"
)

var goRepo *git.Repository

func syncRepo(goRepoDir string) error {
	fmt.Println("syncing repo")
	repo, err := git.PlainOpen(goRepoDir)
	if err != nil {
		// delete the cache file and try again
		err = os.Remove(repoCacheFile)
		if err != nil {
			return err
		}
		return cloneRepo(goRepoDir)
	}

	cmd := exec.Command("git", "-C", goRepoDir, "fetch", "--update-head-ok", "origin", "+refs/*:refs/*")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	goRepo = repo

	return nil
}

func cloneRepo(goRepoDir string) error {
	fmt.Println("cloning repo")
	// Since golang codebase is a huge repo, clone it with go implementation is extremely slow
	cmd := exec.Command("git",
		"clone", "--progress", "--no-checkout",
		"https://github.com/golang/go.git", goRepoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	err = syncRepo(goRepoDir)
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(goRepoDir)
	if err != nil {
		return err
	}
	fmt.Println("opened repo")

	goRepo = r
	return nil
}

const GoRepoEnv = "GORE_GO_REPO"

// for CI cases
func tryLoadFromEnv() error {
	goRepoDir, ok := os.LookupEnv(GoRepoEnv)
	if ok {
		return syncRepo(goRepoDir)
	}
	return errors.New("no env")
}

func init() {
	err := tryLoadFromEnv()
	if err == nil {
		// if we have the repo, we don't need to do anything
		return
	}

	cached, err := os.ReadFile(repoCacheFile)
	if err == nil {
		err := syncRepo(string(cached))
		if err != nil {
			if !errors.Is(err, git.NoErrAlreadyUpToDate) {
				panic(err)
			}
		}
	} else {
		goRepoDir, err := os.MkdirTemp("", "gore-go-repo-")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(repoCacheFile, []byte(goRepoDir), 0644)
		if err != nil {
			panic(err)
		}

		err = cloneRepo(goRepoDir)
		if err != nil {
			panic(err)
		}
	}
}
