// Copyright (C) 2026 GoRE Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package gore

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestITabLinksDataGolden(t *testing.T) {
	for _, version := range []string{"1.26.0", "1.27.0"} {
		for _, target := range []string{"linux-amd64", "linux-386", "windows-amd64", "windows-386", "darwin-amd64", "darwin-arm64"} {
			name := "gold-" + target + "-" + version
			t.Run(name, func(t *testing.T) {
				path := filepath.Join("testdata", "gold", name)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Skip("golden file unavailable")
				}
				f, err := Open(path)
				require.NoError(t, err)
				defer f.Close()
				md, err := f.Moduledata()
				require.NoError(t, err)
				addresses, err := md.ITabLinksData()
				require.NoError(t, err)
				require.NotEmpty(t, addresses)
				if version == "1.26.0" {
					require.Equal(t, md.ITabLinks().Length, uint64(len(addresses)))
				} else {
					require.Zero(t, md.ITabLinks().Length)
					require.Equal(t, f.moduledata.TypesAddr+f.moduledata.ITabOffset, addresses[0])
				}
				for _, addr := range addresses {
					require.NotZero(t, addr)
					_, err := f.Bytes(addr, uint64(2*f.FileInfo.WordSize))
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestGo127ITabCompilerLayout(t *testing.T) {
	compiler := exec.CommandContext(t.Context(), "go", "env", "GOVERSION")
	compiler.Dir = t.TempDir()
	version, err := compiler.CombinedOutput()
	require.NoError(t, err, "%s", version)
	if !usesGo127TypeLayout(strings.TrimSpace(string(version))) {
		t.Skip("fixture requires a Go 1.27+ compiler")
	}
	const source = `package main
type twoMethods interface { First() int; Second() int }
type concrete int
func (v concrete) First() int { return int(v) }
func (v concrete) Second() int { return int(v)+1 }
var itabOnly twoMethods = concrete(9)
func main() { println(itabOnly.First(), itabOnly.Second()) }
`
	for _, target := range []struct {
		os, arch      string
		pie, stripped bool
	}{
		{"linux", "amd64", false, false}, {"linux", "amd64", true, true},
		{"linux", "386", false, false}, {"linux", "s390x", false, false},
		{"linux", "mips", false, false}, {"windows", "amd64", false, false},
		{"darwin", "arm64", true, false}, {"js", "wasm", false, true},
		{"wasip1", "wasm", false, true},
	} {
		name := fmt.Sprintf("%s-%s-pie%t-stripped%t", target.os, target.arch, target.pie, target.stripped)
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			src, bin := filepath.Join(dir, "main.go"), filepath.Join(dir, "binary")
			require.NoError(t, os.WriteFile(src, []byte(source), 0600))
			args := []string{"build", "-gcflags=-S", "-o", bin}
			if target.pie {
				args = append(args, "-buildmode=pie")
			}
			if target.stripped {
				args = append(args, "-ldflags=-s -w")
			}
			args = append(args, src)
			cmd := exec.CommandContext(t.Context(), "go", args...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS="+target.os, "GOARCH="+target.arch, "CGO_ENABLED=0")
			assembly, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", assembly)
			f, err := Open(bin)
			require.NoError(t, err)
			defer f.Close()
			ver, err := f.GetCompilerVersion()
			require.NoError(t, err)
			if !usesGo127TypeLayout(ver.Name) {
				t.Skip("fixture requires a Go 1.27+ compiler")
			}
			md, err := f.Moduledata()
			require.NoError(t, err)
			addresses, err := md.ITabLinksData()
			require.NoError(t, err)
			typs, err := f.GetTypes()
			require.NoError(t, err)
			byAddr := map[uint64]*GoType{}
			for _, typ := range typs {
				byAddr[typ.Addr] = typ
			}
			match := regexp.MustCompile(`(?m)^go:itab.main.concrete,main.twoMethods SRODATA[^\r\n]* size=(\d+)`).FindSubmatch(assembly)
			require.Len(t, match, 2, "compiler's itab size missing")
			size, err := strconv.ParseUint(string(match[1]), 10, 64)
			require.NoError(t, err)
			word := uint64(f.FileInfo.WordSize)
			found := false
			for i, addr := range addresses {
				data, err := f.Bytes(addr, 2*word+4)
				require.NoError(t, err)
				inter := md.ResolvePointer(f.moduledata.itabWord(data), addr)
				concrete := md.ResolvePointer(f.moduledata.itabWord(data[word:]), addr+word)
				require.Contains(t, byAddr, inter)
				require.Contains(t, byAddr, concrete)
				if byAddr[inter].Name != "main.twoMethods" || byAddr[concrete].Name != "main.concrete" {
					continue
				}
				found = true
				end := f.moduledata.TypesAddr + f.moduledata.ITabOffset + f.moduledata.ITabSize
				if i+1 < len(addresses) {
					end = addresses[i+1]
				}
				require.Equal(t, size, end-addr, "encoded record must match compiler layout")
				hash, err := f.Bytes(concrete+2*word, 4)
				require.NoError(t, err)
				require.Equal(t, f.FileInfo.ByteOrder.Uint32(hash), f.FileInfo.ByteOrder.Uint32(data[2*word:]))
				if !target.stripped && target.arch != "wasm" {
					symbol, err := f.GetSymbol("main.itabOnly")
					require.NoError(t, err)
					value, err := f.Bytes(symbol.Value, word)
					require.NoError(t, err)
					require.Equal(t, addr, md.ResolvePointer(f.moduledata.itabWord(value), symbol.Value))
				}
			}
			require.True(t, found, "itab referenced only from the interface value must be recovered")
		})
	}
}

func TestITabLinksDataRejectsInvalidRanges(t *testing.T) {
	info := &FileInfo{WordSize: 8, ByteOrder: binary.LittleEndian, goversion: &GoVersion{Name: "go1.26.0"}}
	fh := &mockFileHandler{mGetSectionDataFromAddress: func(uint64) (uint64, []byte, error) { return 100, make([]byte, 16), nil }}
	for _, md := range []moduledata{
		{ITabLinkAddr: 100, ITabLinkLen: ^uint64(0)},
		{ITabLinkAddr: ^uint64(0) - 3, ITabLinkLen: 1},
		{ITabLinkAddr: 99, ITabLinkLen: 1},
		{ITabLinkAddr: 108, ITabLinkLen: 2},
	} {
		md.fileInfo = info
		md.fh = fh
		require.NotPanics(t, func() { addresses, err := md.ITabLinksData(); require.Error(t, err); require.Nil(t, addresses) })
	}
	_, err := (moduledata{}).ITabLinksData()
	require.ErrorIs(t, err, ErrNoGoVersionFound)
}
