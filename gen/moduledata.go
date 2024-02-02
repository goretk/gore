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
	"go/ast"
	"go/format"
	"go/parser"
	"golang.org/x/mod/semver"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var moduleDataMatcher = regexp.MustCompile(`(?m:type moduledata struct {[^}]+})`)

// generateModuleDataSources
// returns a map of moduledata sources for each go version, from 1.5 to the latest we know so far.
func getModuleDataSources() (map[int]string, error) {
	ret := make(map[int]string)

	f, err := os.OpenFile(goversionCsv, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return nil, fmt.Errorf("error when opening goversions.csv: %w", err)
	}
	knownVersion, err := getCsvStoredGoversions(f)
	if err != nil {
		return nil, fmt.Errorf("error when getting stored go versions: %w", err)
	}

	knownVersionSlice := make([]string, 0, len(knownVersion))

	matcher := regexp.MustCompile(`[a-zA-Z]`)
	for ver := range knownVersion {
		// rc/beta version not in consideration
		if matcher.MatchString(ver[2:]) {
			continue
		}

		knownVersionSlice = append(knownVersionSlice, semver.MajorMinor("v"+strings.TrimPrefix(ver, "go")))
	}
	semver.Sort(knownVersionSlice)

	latest := knownVersionSlice[len(knownVersionSlice)-1]

	maxMinor, err := strconv.Atoi(strings.Split(latest, ".")[1])
	if err != nil {
		return nil, fmt.Errorf("error when getting latest go version: %w, %s", err, latest)
	}

	for i := 5; i <= maxMinor; i++ {
		fmt.Println("Fetching moduledata for go1." + strconv.Itoa(i) + "...")
		branch := fmt.Sprintf("release-branch.go1.%d", i)
		contents, _, _, err := githubClient.Repositories.GetContents(
			context.Background(),
			"golang", "go",
			"src/runtime/symtab.go",
			&github.RepositoryContentGetOptions{Ref: branch})
		if err != nil {
			return nil, err
		}

		content, err := contents.GetContent()
		if err != nil {
			return nil, err
		}

		structStr := moduleDataMatcher.FindString(content)
		if structStr == "" {
			return nil, fmt.Errorf("moduledata struct not found in symtab.go")
		}

		// make it an expression for further parse
		structStr = strings.TrimPrefix(structStr, "type moduledata ")

		ret[i] = structStr
	}

	return ret, nil
}

type moduleDataGenerator struct {
	buf *bytes.Buffer

	knownVersions []int
}

func (g *moduleDataGenerator) init() {
	g.buf = &bytes.Buffer{}
	g.buf.WriteString(moduleDataHeader)
}

func (g *moduleDataGenerator) add(versionCode int, code string) error {
	g.knownVersions = append(g.knownVersions, versionCode)

	err := g.writeVersionedModuleData(versionCode, code)
	if err != nil {
		return err
	}

	return nil
}

func (g *moduleDataGenerator) writeSelector() {
	g.writeln("func selectModuleData(v int, bits int) (modulable,error) {")
	g.writeln("switch {")

	for _, versionCode := range g.knownVersions {
		for _, bits := range []int{32, 64} {
			g.writeln("case v == %d && bits == %d:", versionCode, bits)
			g.writeln("return &%s{}, nil", g.generateTypeName(versionCode, bits))
		}
	}
	g.writeln("default:")
	g.writeln(`return nil, fmt.Errorf("unsupported version %%d and bits %%d", v, bits)`)

	g.writeln("}\n}\n")

}

func (*moduleDataGenerator) generateTypeName(versionCode int, bits int) string {
	return fmt.Sprintf("moduledata_1_%d_%d", versionCode, bits)
}

func (*moduleDataGenerator) wrapValue(name string, bits int) string {
	if bits == 32 {
		return fmt.Sprintf("uint64(%s)", name)
	}
	return name
}

func (*moduleDataGenerator) title(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *moduleDataGenerator) writeln(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(g.buf, format+"\n", a...)
}

func (g *moduleDataGenerator) writeVersionedModuleData(versionCode int, code string) error {
	expr, err := parser.ParseExpr(code)
	if err != nil {
		return fmt.Errorf("failed to parse moduledata expression: %w", err)
	}
	structExpr, ok := expr.(*ast.StructType)
	if !ok {
		return fmt.Errorf("failed to parse moduledata expression")
	}

	writeCode := func(bits int) {
		g.writeln("type %s struct {\n", g.generateTypeName(versionCode, bits))

		knownFields := make(map[string]struct{})
	search:
		for _, field := range structExpr.Fields.List {
			if len(field.Names) == 0 {
				// skip anonymous field
				// currently only sys.NotInHeap
				continue
			}

			for _, name := range field.Names {
				if name.Name == "modulename" {
					// no more data needed
					break search
				}
				knownFields[name.Name] = struct{}{}

				switch t := field.Type.(type) {
				case *ast.StarExpr:
					g.writeln("%s uint%d", g.title(name.Name), bits)
				case *ast.ArrayType:
					g.writeln("%s, %[1]slen, %[1]scap uint%d", g.title(name.Name), bits)
				case *ast.Ident:
					switch t.Name {
					case "uintptr":
						g.writeln("%s uint%d", g.title(name.Name), bits)
					case "string":
						g.writeln("%s, %[1]slen uint%d", g.title(name.Name), bits)
					case "uint8":
						g.writeln("%s uint8", g.title(name.Name))
					default:
						panic(fmt.Sprintf("unhandled type: %+v", t))
					}
				default:
					panic(fmt.Sprintf("unhandled type: %+v", t))
				}
			}
		}

		g.writeln("}\n\n")

		// generate toModuledata method
		exist := func(name ...string) bool {
			for _, n := range name {
				if _, ok := knownFields[n]; !ok {
					return false
				}
			}
			return true
		}

		g.writeln("func (md %s) toModuledata() moduledata {", g.generateTypeName(versionCode, bits))
		g.writeln("return moduledata{")

		if exist("text", "etext") {
			g.writeln("TextAddr: %s,", g.wrapValue("md.Text", bits))
			g.writeln("TextLen: %s,", g.wrapValue("md.Etext - md.Text", bits))
		}

		if exist("noptrdata", "enoptrdata") {
			g.writeln("NoPtrDataAddr: %s,", g.wrapValue("md.Noptrdata", bits))
			g.writeln("NoPtrDataLen: %s,", g.wrapValue("md.Enoptrdata - md.Noptrdata", bits))
		}

		if exist("data", "edata") {
			g.writeln("DataAddr: %s,", g.wrapValue("md.Data", bits))
			g.writeln("DataLen: %s,", g.wrapValue("md.Edata - md.Data", bits))
		}

		if exist("bss", "ebss") {
			g.writeln("BssAddr: %s,", g.wrapValue("md.Bss", bits))
			g.writeln("BssLen: %s,", g.wrapValue("md.Ebss - md.Bss", bits))
		}

		if exist("noptrbss", "enoptrbss") {
			g.writeln("NoPtrBssAddr: %s,", g.wrapValue("md.Noptrbss", bits))
			g.writeln("NoPtrBssLen: %s,", g.wrapValue("md.Enoptrbss - md.Noptrbss", bits))
		}

		if exist("types", "etypes") {
			g.writeln("TypesAddr: %s,", g.wrapValue("md.Types", bits))
			g.writeln("TypesLen: %s,", g.wrapValue("md.Etypes - md.Types", bits))
		}

		if exist("typelinks") {
			g.writeln("TypelinkAddr: %s,", g.wrapValue("md.Typelinks", bits))
			g.writeln("TypelinkLen: %s,", g.wrapValue("md.Typelinkslen", bits))
		}

		if exist("itablinks") {
			g.writeln("ITabLinkAddr: %s,", g.wrapValue("md.Itablinks", bits))
			g.writeln("ITabLinkLen: %s,", g.wrapValue("md.Itablinkslen", bits))
		}

		if exist("ftab") {
			g.writeln("FuncTabAddr: %s,", g.wrapValue("md.Ftab", bits))
			g.writeln("FuncTabLen: %s,", g.wrapValue("md.Ftablen", bits))
		}

		if exist("pclntable") {
			g.writeln("PCLNTabAddr: %s,", g.wrapValue("md.Pclntable", bits))
			g.writeln("PCLNTabLen: %s,", g.wrapValue("md.Pclntablelen", bits))
		}

		if exist("gofunc") {
			g.writeln("GoFuncVal: %s,", g.wrapValue("md.Gofunc", bits))
		}

		g.writeln("}\n}\n")
	}

	writeCode(32)
	writeCode(64)

	return nil
}

func generateModuleData() {
	sources, err := getModuleDataSources()
	if err != nil {
		panic(err)
	}

	g := moduleDataGenerator{}
	g.init()

	versionCodes := make([]int, 0, len(sources))
	for versionCode := range sources {
		versionCodes = append(versionCodes, versionCode)
	}

	sort.Ints(versionCodes)

	for _, versionCode := range versionCodes {
		err = g.add(versionCode, sources[versionCode])
		if err != nil {
			panic(err)
		}
	}

	g.writeSelector()

	out, err := format.Source(g.buf.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(moduleDataOutputFile, out, 0o666)
	if err != nil {
		panic(err)
	}
}
