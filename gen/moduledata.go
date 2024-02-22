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
	"cmp"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goretk/gore/extern"
	"github.com/goretk/gore/extern/gover"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func getMaxVersionBit() (int, error) {
	f, err := os.OpenFile(goversionCsv, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return 0, fmt.Errorf("error when opening goversions.csv: %w", err)
	}
	knownVersion, err := getCsvStoredGoversions(f)
	if err != nil {
		return 0, fmt.Errorf("error when getting stored go versions: %w", err)
	}

	knownVersionSlice := make([]string, 0, len(knownVersion))

	matcher := regexp.MustCompile(`[a-zA-Z]`)
	for ver := range knownVersion {
		// rc/beta version not in consideration
		if matcher.MatchString(ver[2:]) {
			continue
		}

		knownVersionSlice = append(knownVersionSlice, extern.StripGo(ver))
	}
	sort.Slice(knownVersionSlice, func(i, j int) bool {
		return gover.Compare(knownVersionSlice[i], knownVersionSlice[j]) < 0
	})

	latest := knownVersionSlice[len(knownVersionSlice)-1]

	maxMinor, err := strconv.Atoi(strings.Split(latest, ".")[1])
	if err != nil {
		return 0, fmt.Errorf("error when getting latest go version: %w, %s", err, latest)
	}
	return maxMinor, nil
}

var moduleNameMatcher = regexp.MustCompile(`moduledata_1_(\d+)_(?:32|64)`)

func getCurrentMaxGoBit() (int, error) {
	contents, err := os.ReadFile(moduleDataOutputFile)
	if err != nil {
		return 0, fmt.Errorf("error when reading moduledata.go: %w", err)
	}

	file, err := parser.ParseFile(token.NewFileSet(), "", contents, 0)
	if err != nil {
		return 0, err
	}

	currentVersion := 5 // ignore version <= 1.5

	// get all struct type names
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						name := typeSpec.Name.Name
						if moduleNameMatcher.MatchString(name) {
							matches := moduleNameMatcher.FindStringSubmatch(name)
							if len(matches) != 2 {
								return 0, fmt.Errorf("error when parsing moduledata.go, matches: %v", matches)
							}

							version, err := strconv.Atoi(matches[1])
							if err != nil {
								return 0, fmt.Errorf("error when parsing moduledata.go, version: %v", matches[1])
							}

							currentVersion = max(currentVersion, version)
						}
					}
				}
			}
		}
	}
	return currentVersion, nil
}

var moduleDataMatcher = regexp.MustCompile(`(?m:type moduledata struct {[^}]+})`)

// generateModuleDataSources
// returns a map of moduledata sources for each go version, from 1.5 to the latest we know so far.
func getModuleDataSources() (map[int]string, error) {
	ret := make(map[int]string)

	maxMinor, err := getMaxVersionBit()
	if err != nil {
		return nil, err
	}

	currentMaxMinor, err := getCurrentMaxGoBit()
	if err != nil {
		return nil, err
	}

	if currentMaxMinor == maxMinor {
		return nil, nil
	}

	for i := 5; i <= maxMinor; i++ {
		fmt.Println("Process moduledata for go1." + strconv.Itoa(i) + "...")
		branch := fmt.Sprintf("release-branch.go1.%d", i)

		// find the tree blob
		reference, err := goRepo.Reference(plumbing.NewBranchReferenceName(branch), false)
		if err != nil {
			return nil, err
		}

		commit, err := goRepo.CommitObject(reference.Hash())
		if err != nil {
			return nil, err
		}

		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}

		symtab, err := tree.File("src/runtime/symtab.go")
		if err != nil {
			return nil, err
		}

		content, err := symtab.Contents()
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

func (g *moduleDataGenerator) writeModuleListGetter() {
	g.writeln("func getModuleDataList(bits int) ([]modulable, error) {")
	g.writeln("if bits != 32 && bits != 64 { return nil, fmt.Errorf(\"unsupported bits %d\", bits)}")

	g.writeln("if bits == 32 {")
	g.writeln("return []modulable{")
	for _, versionCode := range g.knownVersions {
		g.writeln("&%s{},", g.generateTypeName(versionCode, 32))
	}
	g.writeln("}, nil")
	g.writeln("} else {")
	g.writeln("return []modulable{")
	for _, versionCode := range g.knownVersions {
		g.writeln("&%s{},", g.generateTypeName(versionCode, 64))
	}
	g.writeln("}, nil")
	g.writeln("}")
	g.writeln("}\n")

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
					g.writeln("%s uint%d", name.Name, bits)
				case *ast.ArrayType:
					g.writeln("%s, %[1]slen, %[1]scap uint%d", name.Name, bits)
				case *ast.Ident:
					switch t.Name {
					case "uintptr":
						g.writeln("%s uint%d", name.Name, bits)
					case "string":
						g.writeln("%s, %[1]slen uint%d", name.Name, bits)
					case "uint8":
						g.writeln("%s uint8", name.Name)
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
			g.writeln("TextAddr: %s,", g.wrapValue("md.text", bits))
			g.writeln("TextLen: %s,", g.wrapValue("md.etext - md.text", bits))
		}

		if exist("noptrdata", "enoptrdata") {
			g.writeln("NoPtrDataAddr: %s,", g.wrapValue("md.noptrdata", bits))
			g.writeln("NoPtrDataLen: %s,", g.wrapValue("md.enoptrdata - md.noptrdata", bits))
		}

		if exist("data", "edata") {
			g.writeln("DataAddr: %s,", g.wrapValue("md.data", bits))
			g.writeln("DataLen: %s,", g.wrapValue("md.edata - md.data", bits))
		}

		if exist("bss", "ebss") {
			g.writeln("BssAddr: %s,", g.wrapValue("md.bss", bits))
			g.writeln("BssLen: %s,", g.wrapValue("md.ebss - md.bss", bits))
		}

		if exist("noptrbss", "enoptrbss") {
			g.writeln("NoPtrBssAddr: %s,", g.wrapValue("md.noptrbss", bits))
			g.writeln("NoPtrBssLen: %s,", g.wrapValue("md.enoptrbss - md.noptrbss", bits))
		}

		if exist("types", "etypes") {
			g.writeln("TypesAddr: %s,", g.wrapValue("md.types", bits))
			g.writeln("TypesLen: %s,", g.wrapValue("md.etypes - md.types", bits))
		}

		if exist("typelinks") {
			g.writeln("TypelinkAddr: %s,", g.wrapValue("md.typelinks", bits))
			g.writeln("TypelinkLen: %s,", g.wrapValue("md.typelinkslen", bits))
		}

		if exist("itablinks") {
			g.writeln("ITabLinkAddr: %s,", g.wrapValue("md.itablinks", bits))
			g.writeln("ITabLinkLen: %s,", g.wrapValue("md.itablinkslen", bits))
		}

		if exist("ftab") {
			g.writeln("FuncTabAddr: %s,", g.wrapValue("md.ftab", bits))
			g.writeln("FuncTabLen: %s,", g.wrapValue("md.ftablen", bits))
		}

		if exist("pclntable") {
			g.writeln("PCLNTabAddr: %s,", g.wrapValue("md.pclntable", bits))
			g.writeln("PCLNTabLen: %s,", g.wrapValue("md.pclntablelen", bits))
		}

		if exist("gofunc") {
			g.writeln("GoFuncVal: %s,", g.wrapValue("md.gofunc", bits))
		}

		g.writeln("}\n}\n")
	}

	writeCode(32)
	writeCode(64)

	return nil
}

func generateModuleData() {
	fmt.Println("Generating " + moduleDataOutputFile)

	sources, err := getModuleDataSources()
	if err != nil {
		panic(err)
	}
	if sources == nil {
		fmt.Println("No need to update " + moduleDataOutputFile)
		return
	}

	g := moduleDataGenerator{}
	g.init()

	versionCodes := make([]int, 0, len(sources))
	for versionCode := range sources {
		versionCodes = append(versionCodes, versionCode)
	}

	slices.SortFunc(versionCodes, func(a, b int) int {
		return -cmp.Compare(a, b)
	})

	for _, versionCode := range versionCodes {
		err = g.add(versionCode, sources[versionCode])
		if err != nil {
			panic(err)
		}
	}

	g.writeSelector()
	g.writeModuleListGetter()

	out, err := format.Source(g.buf.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(moduleDataOutputFile, out, 0o666)
	if err != nil {
		panic(err)
	}
}
