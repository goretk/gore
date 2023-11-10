// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
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

package gore

import (
	"debug/gosym"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	resourceFolder = "testdata"
	fixedBuildID   = "DrtsigZmOidE-wfbFVNF/io-X8KB-ByimyyODdYUe/Z7tIlu8GbOwt0Jup-Hji/fofocVx5sk8UpaKMTx0a"
)

func TestIssue11NoNoteSectionELF(t *testing.T) {
	// Build test resource
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("No go tool chain found: " + err.Error())
	}
	tmpdir, err := ioutil.TempDir("", "TestGORE-Issue11")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)
	src := filepath.Join(tmpdir, "a.go")
	err = ioutil.WriteFile(src, []byte(testresourcesrc), 0644)
	if err != nil {
		panic(err)
	}
	exe := filepath.Join(tmpdir, "a")
	args := []string{"build", "-o", exe, "-ldflags", "-s -w -buildid=", src}
	cmd := exec.Command(goBin, args...)
	gopatch := os.Getenv("GOPATH")
	if gopatch == "" {
		gopatch = tmpdir
	}
	cmd.Env = append(cmd.Env, "GOCACHE="+tmpdir, "GOOS=linux", "GOPATH="+gopatch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic("building test executable failed: " + string(out))
	}

	_, err = Open(exe)
	assert.NoError(t, err, "Should not fail to open an ELF file without a notes section.")
}

func TestGoldFiles(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}

	mustParse := func(n int, e error) int {
		if err != nil {
			t.Fatalf("parsing error: %s", err)
		}
		return n
	}

	for _, file := range goldFiles {
		t.Run("compiler_version_"+file, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// TODO: Remove this check when arm support has been added.
			if strings.Contains(file, "arm64") {
				t.Skip("ARM currently not supported")
			}

			// Loading resource
			resource, err := getGoldTestResourcePath(file)
			require.NoError(err)
			f, err := Open(resource)
			require.NoError(err)
			require.NotNil(f)

			// Get info from filename gold-os-arch-goversion
			fileInfo := strings.Split(file, "-")

			// If patch level is 0, it is dropped. For example. 10.0.0 is 10.0.
			// This was changed in 1.21 so if the version is 1.21 or greater, we take
			// the whole string.
			var actualVersion string
			verArr := strings.Split(fileInfo[3], ".")
			if len(verArr) == 3 && verArr[2] == "0" && mustParse(strconv.Atoi(verArr[1])) < 21 {
				actualVersion = strings.Join(verArr[:2], ".")
			} else {
				actualVersion = fileInfo[3]
			}

			// Tests
			// Not in 1.2 and 1.3
			if strings.HasPrefix(actualVersion, "1.2.") ||
				strings.HasPrefix(actualVersion, "1.3.") ||
				actualVersion == "1.2" || actualVersion == "1.3" {
				t.SkipNow()
			}
			version, err := f.GetCompilerVersion()
			assert.NoError(err)
			require.NotNil(version, "Version should not be nil")
			assert.Equal("go"+actualVersion, version.Name, "Incorrect version for "+file)

			// Clean up
			f.Close()
		})
	}
}

func TestSetGoVersion(t *testing.T) {
	assert := assert.New(t)

	t.Run("right error on wrong version string", func(t *testing.T) {
		f := new(GoFile)
		f.FileInfo = new(FileInfo)

		err := f.SetGoVersion("invalid version string")

		assert.Error(err, "Should return an error when the version string is invalid")
		assert.Equal(ErrInvalidGoVersion, err, "Incorrect error value returned")
	})

	t.Run("should set correct version", func(t *testing.T) {
		versionStr := "go1.12"
		expected := goversions[versionStr]
		f := new(GoFile)
		f.FileInfo = new(FileInfo)

		err := f.SetGoVersion(versionStr)

		assert.Nil(err, "Should not return an error when the version string is correct format")
		assert.Equal(expected, f.FileInfo.goversion, "Incorrect go version has be set")
	})
}

type mockFileHandler struct {
	mGetSectionDataFromOffset func(uint64) (uint64, []byte, error)
}

func (m *mockFileHandler) getFile() *os.File {
	panic("not implemented")
}

func (m *mockFileHandler) Close() error {
	panic("not implemented")
}

func (m *mockFileHandler) getPCLNTab() (*gosym.Table, error) {
	panic("not implemented")
}

func (m *mockFileHandler) getRData() ([]byte, error) {
	panic("not implemented")
}

func (m *mockFileHandler) getCodeSection() ([]byte, error) {
	panic("not implemented")
}

func (m *mockFileHandler) getSectionDataFromOffset(o uint64) (uint64, []byte, error) {
	return m.mGetSectionDataFromOffset(o)
}

func (m *mockFileHandler) getSectionData(string) (uint64, []byte, error) {
	panic("not implemented")
}

func (m *mockFileHandler) getFileInfo() *FileInfo {
	panic("not implemented")
}

func (m *mockFileHandler) getPCLNTABData() (uint64, []byte, error) {
	panic("not implemented")
}

func (m *mockFileHandler) moduledataSection() string {
	panic("not implemented")
}

func (m *mockFileHandler) getBuildID() (string, error) {
	panic("not implemented")
}

func TestBytes(t *testing.T) {
	assert := assert.New(t)
	expectedBase := uint64(0x40000)
	expectedSection := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}
	expectedBytes := []byte{0x2, 0x3, 0x4, 0x5}
	address := uint64(expectedBase + 2)
	length := uint64(len(expectedBytes))
	fh := &mockFileHandler{
		mGetSectionDataFromOffset: func(a uint64) (uint64, []byte, error) {
			if a > expectedBase+uint64(len(expectedSection)) || a < expectedBase {
				return 0, nil, errors.New("out of bound")
			}
			return expectedBase, expectedSection, nil
		},
	}
	f := &GoFile{fh: fh}

	data, err := f.Bytes(address, length)
	assert.NoError(err, "Should not return an error")
	assert.Equal(expectedBytes, data, "Return data not as expected")
}

func getTestResourcePath(resource string) (string, error) {
	return filepath.Abs(filepath.Join(resourceFolder, resource))
}

func getGoldTestResourcePath(resource string) (string, error) {
	return filepath.Abs(filepath.Join(resourceFolder, "gold", resource))
}

func getGoldenResources() ([]string, error) {
	folderPath, err := filepath.Abs(resourceFolder)
	if err != nil {
		return nil, err
	}
	folder, err := ioutil.ReadDir(filepath.Join(folderPath, "gold"))
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range folder {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "gold-") {
			continue
		}
		files = append(files, f.Name())
	}
	return files, nil
}

const testresourcesrc = `
package main

//go:noinline
func getData() string {
	return "Name: GoRE"
}

func main() {
	data := getData()
	data += " | Test"
}
`
