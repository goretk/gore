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
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTypes(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}
	for _, test := range goldFiles {
		t.Run("get_types_"+test, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			fp, err := getTestResourcePath("gold/" + test)
			require.NoError(err, "Failed to get path to resource")
			if _, err = os.Stat(fp); os.IsNotExist(err) {
				// Skip this file because it doesn't exist
				// t.Skip will cause the parent test to be skipped.
				fmt.Printf("[SKIPPING TEST] golden fille %s does not exist\n", test)
				return
			}
			f, err := Open(fp)
			require.NoError(err, "Failed to get path to the file")
			defer f.Close()

			typs, err := f.GetTypes()
			require.NoError(err, "Should parse with no error")

			var simpleStructTested bool
			var complexStructTested bool
			var stringerInterfaceTested bool
			for _, typ := range typs {
				if typ.Name == "fmt.Stringer" && typ.PackagePath == "fmt" &&
					GoVersionCompare(f.FileInfo.goversion.Name, "go1.7beta1") >= 0 {
					assert.Equal(reflect.Interface, typ.Kind, "Stringer should be an interface")
					assert.Len(typ.Methods, 1, "Stringer should have 1 function defined")
					assert.Equal("String", typ.Methods[0].Name, "Stringer's function should have the name of String")

					stringerInterfaceTested = true
				}
				if typ.Name == "main.simpleStruct" && typ.PackagePath == "main" {
					assert.Equal(reflect.Struct, typ.Kind, "simpleStruct parsed as wrong type")
					assert.Len(typ.Fields, 2, "simpleStruct should have 2 fields")

					// Checking fields first should be a string and second an int
					assert.Equal(reflect.String, typ.Fields[0].Kind, "First field is the wrong kind.")
					assert.Equal("name", typ.Fields[0].FieldName, "First field has the wrong name.")

					assert.Equal(reflect.Int, typ.Fields[1].Kind, "Second field is the wrong kind.")
					assert.Equal("age", typ.Fields[1].FieldName, "Second field has the wrong name.")

					simpleStructTested = true
				}

				if typ.Name == "main.myComplexStruct" && typ.PackagePath == "main" &&
					GoVersionCompare(f.FileInfo.goversion.Name, "go1.7beta1") >= 0 {
					assert.Equal(reflect.Struct, typ.Kind, "myComplexStruct parsed as wrong type")
					assert.Len(typ.Fields, 8, "myComplexStruct should have 7 fields")

					// Checking fields first should be a string and second an int
					assert.Equal(reflect.String, typ.Fields[0].Kind, "First field is the wrong kind.")
					assert.Equal("MyString", typ.Fields[0].FieldName, "First field has the wrong name.")
					assert.Equal(`json:"String"`, typ.Fields[0].FieldTag, "Field tag incorrectly parsed")

					assert.Equal(reflect.Ptr, typ.Fields[1].Kind, "Second field is the wrong kind.")
					assert.Equal("person", typ.Fields[1].FieldName, "Second field has the wrong name.")
					assert.Equal(reflect.Struct, typ.Fields[1].Element.Kind, "Second field resolves to the wrong kind.")

					assert.Len(typ.Fields[1].Element.Fields, 2, "simpleStruct should have 2 fields")
					assert.Equal(reflect.String, typ.Fields[1].Element.Fields[0].Kind, "First resolved field is the wrong kind.")
					assert.Equal("name", typ.Fields[1].Element.Fields[0].FieldName, "First resolved field has the wrong name.")

					assert.Equal(reflect.Int, typ.Fields[1].Element.Fields[1].Kind, "Second resolved field is the wrong kind.")
					assert.Equal("age", typ.Fields[1].Element.Fields[1].FieldName, "Second resolved field has the wrong name.")

					// Methods on simpleStruct
					assert.Len(typ.Fields[1].Methods, 1, "simpleStruct should have 1 method")
					assert.Equal("String", typ.Fields[1].Methods[0].Name, "Wrong method name")

					// Checking other types
					assert.Equal(reflect.Array, typ.Fields[2].Kind, "Third field is the wrong kind.")
					assert.Equal("myArray", typ.Fields[2].FieldName, "Third field has the wrong name.")
					assert.Equal(2, typ.Fields[2].Length, "Array length is wrong")
					assert.Equal(reflect.Int, typ.Fields[2].Element.Kind, "Array element is wrong")

					assert.Equal(reflect.Slice, typ.Fields[3].Kind, "4th field is the wrong kind.")
					assert.Equal("mySlice", typ.Fields[3].FieldName, "4th field has the wrong name.")
					assert.Equal(reflect.Uint, typ.Fields[3].Element.Kind, "Slice element is wrong")

					assert.Equal(reflect.Chan, typ.Fields[4].Kind, "5th field is the wrong kind.")
					assert.Equal("myChan", typ.Fields[4].FieldName, "5th field has the wrong name.")
					assert.Equal(reflect.Struct, typ.Fields[4].Element.Kind, "Chan element is wrong")
					assert.Equal(ChanBoth, typ.Fields[4].ChanDir, "Chan direction is wrong")

					assert.Equal(reflect.Map, typ.Fields[5].Kind, "6th field is the wrong kind.")
					assert.Equal("myMap", typ.Fields[5].FieldName, "6th field has the wrong name.")
					assert.Equal(reflect.String, typ.Fields[5].Key.Kind, "Map key is wrong")
					assert.Equal(reflect.Int, typ.Fields[5].Element.Kind, "Map element is wrong")

					assert.Equal(reflect.Func, typ.Fields[6].Kind, "7th field is the wrong kind.")
					assert.Equal("myFunc", typ.Fields[6].FieldName, "7th field has the wrong name.")
					assert.Equal(reflect.String, typ.Fields[6].FuncArgs[0].Kind, "Function argument kind is wrong.")
					assert.Equal(reflect.Int, typ.Fields[6].FuncArgs[1].Kind, "Function argument kind is wrong.")
					assert.Equal(reflect.Uint, typ.Fields[6].FuncReturnVals[0].Kind, "Function return kind is wrong.")

					// Embedded struct
					assert.True(typ.Fields[7].FieldAnon, "Last field should be an anonymous struct")
					assert.Equal(reflect.Struct, typ.Fields[7].Kind, "Last field should be an anonymous struct")
					assert.Equal("val", typ.Fields[7].Fields[0].FieldName, "Last field's field should be called val")

					complexStructTested = true
				}

				if typ.Name == "cpu.option" && typ.PackagePath == "" &&
					GoVersionCompare(f.FileInfo.goversion.Name, "go1.7beta1") >= 0 {
					for _, field := range typ.Fields {
						assert.Equal("", field.FieldTag, "Field Tag should be empty")
					}
				}
			}
			if GoVersionCompare(f.FileInfo.goversion.Name, "go1.7beta1") >= 0 {
				assert.True(complexStructTested, "myComplexStruct was not found")
				assert.True(stringerInterfaceTested, "fmt.Stringer was not found")
			}
			assert.True(simpleStructTested, "simpleStruct was not found")
		})
	}
}

func TestGoTypeStringer(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		typ      *GoType
		expected string
	}{
		{&GoType{Kind: reflect.String}, "string"},
		{&GoType{Kind: reflect.Bool}, "bool"},
		{&GoType{Kind: reflect.Float32}, "float32"},
		{&GoType{Kind: reflect.Float64}, "float64"},
		{&GoType{Kind: reflect.Int}, "int"},
		{&GoType{Kind: reflect.Int8}, "int8"},
		{&GoType{Kind: reflect.Int16}, "int16"},
		{&GoType{Kind: reflect.Int32}, "int32"},
		{&GoType{Kind: reflect.Int64}, "int64"},
		{&GoType{Kind: reflect.Uint}, "uint"},
		{&GoType{Kind: reflect.Uint8}, "uint8"},
		{&GoType{Kind: reflect.Uint16}, "uint16"},
		{&GoType{Kind: reflect.Uint32}, "uint32"},
		{&GoType{Kind: reflect.Uint64}, "uint64"},
		{&GoType{Kind: reflect.Slice, Element: &GoType{Kind: reflect.Int}}, "[]int"},
		{&GoType{Kind: reflect.Array, Element: &GoType{Kind: reflect.Uint}, Length: 10}, "[10]uint"},
		{&GoType{Kind: reflect.Map, Element: &GoType{Kind: reflect.Uint}, Key: &GoType{Kind: reflect.String}}, "map[string]uint"},
		{&GoType{Kind: reflect.Struct, Name: "testStruct"}, "testStruct"},
		{&GoType{Kind: reflect.Struct}, "struct{}"},
		{&GoType{Kind: reflect.Ptr, Element: &GoType{Kind: reflect.Struct, Name: "testStruct"}}, "*testStruct"},
		{&GoType{Kind: reflect.Chan, Element: &GoType{Kind: reflect.Struct}}, "chan struct{}"},
		{&GoType{Kind: reflect.Chan, ChanDir: ChanBoth, Element: &GoType{Kind: reflect.Struct}}, "chan struct{}"},
		{&GoType{Kind: reflect.Chan, ChanDir: ChanRecv, Element: &GoType{Kind: reflect.Struct}}, "<-chan struct{}"},
		{&GoType{Kind: reflect.Chan, ChanDir: ChanSend, Element: &GoType{Kind: reflect.Struct}}, "chan<- struct{}"},
		{&GoType{
			Kind:           reflect.Func,
			FuncArgs:       []*GoType{{Kind: reflect.String}, {Kind: reflect.Int}},
			FuncReturnVals: []*GoType{{Kind: reflect.Uint}},
		}, "func(string, int) uint"},
		{&GoType{
			Kind:           reflect.Func,
			FuncArgs:       []*GoType{{Kind: reflect.String}, {Kind: reflect.Int}},
			FuncReturnVals: []*GoType{{Kind: reflect.Uint}, {Kind: reflect.Struct}},
		}, "func(string, int) (uint, struct{})"},
		{&GoType{
			Kind:     reflect.Func,
			FuncArgs: []*GoType{{Kind: reflect.String}},
		}, "func(string)"},
		{&GoType{
			Kind:           reflect.Func,
			FuncReturnVals: []*GoType{{Kind: reflect.Uint}},
		}, "func() uint"},
		{&GoType{
			Kind: reflect.Func,
		}, "func()"},
		{&GoType{Kind: reflect.Interface, Name: "fmt.Stringer"}, "fmt.Stringer"},
		{&GoType{Kind: reflect.Interface, Name: "error"}, "error"},
		{&GoType{Kind: reflect.Interface}, "interface{}"},
	}
	for _, test := range tests {
		assert.Equal(test.expected, test.typ.String())
	}
}

func TestStructDef(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		typ      *GoType
		expected string
	}{
		{&GoType{Kind: reflect.String}, ""},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myStruct",
		}, "type myStruct struct{}"},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myComplexStruct",
			Fields: []*GoType{
				{FieldName: "myString", Kind: reflect.String},
				{FieldName: "person", Kind: reflect.Ptr, Element: &GoType{Kind: reflect.Struct, Name: "simpleStruct"}},
				{FieldName: "myArray", Kind: reflect.Array, Length: 2, Element: &GoType{Kind: reflect.Int}},
				{FieldName: "mySlice", Kind: reflect.Slice, Element: &GoType{Kind: reflect.Uint}},
				{FieldName: "myChan", Kind: reflect.Chan, Element: &GoType{Kind: reflect.Struct}},
				{FieldName: "myMap", Kind: reflect.Map, Element: &GoType{Kind: reflect.Int}, Key: &GoType{Kind: reflect.String}},
				{FieldName: "myFunc", Kind: reflect.Func, FuncArgs: []*GoType{{Kind: reflect.String}, {Kind: reflect.Int}}, FuncReturnVals: []*GoType{{Kind: reflect.Uint}}},
			}}, complexStructDef},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myComplexStruct",
			Fields: []*GoType{
				{FieldName: "myString", Kind: reflect.String},
				{FieldName: "person", Kind: reflect.Ptr, Element: &GoType{Kind: reflect.Struct, Name: "simpleStruct"}},
				{FieldName: "myArray", Kind: reflect.Array, Length: 2, Element: &GoType{Kind: reflect.Int}},
				{FieldName: "mySlice", Kind: reflect.Slice, Element: &GoType{Kind: reflect.Uint}},
				{FieldName: "myChan", Kind: reflect.Chan, Element: &GoType{Kind: reflect.Struct}},
				{FieldName: "myMap", Kind: reflect.Map, Element: &GoType{Kind: reflect.Int}, Key: &GoType{Kind: reflect.String}},
				{FieldName: "myFunc", Kind: reflect.Func, FuncArgs: []*GoType{{Kind: reflect.String}, {Kind: reflect.Int}}, FuncReturnVals: []*GoType{{Kind: reflect.Uint}}},
				{FieldAnon: true, Kind: reflect.Struct, Name: "embeddedType"},
			}}, complexStructWithAnonDef},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myStruct",
			Fields: []*GoType{
				{FieldName: "myString", Kind: reflect.String, FieldTag: `json:"String"`},
			}}, structWithFieldTag},
	}
	for _, test := range tests {
		assert.Equal(test.expected, StructDef(test.typ))
	}
}

func TestMethodDefsAll(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		typ      *GoType
		expected string
	}{
		{&GoType{Kind: reflect.String}, ""},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myStruct",
			Methods: []*TypeMethod{
				{Name: "Read", Type: &GoType{
					Kind:           reflect.Func,
					FuncArgs:       []*GoType{{Kind: reflect.Slice, Element: &GoType{Kind: reflect.Int8}}},
					FuncReturnVals: []*GoType{{Kind: reflect.Int}, {Kind: reflect.Interface, Name: "error"}}}},
				{Name: "Close", Type: &GoType{
					Kind:           reflect.Func,
					FuncReturnVals: []*GoType{{Kind: reflect.Interface, Name: "error"}}}},
				{Name: "private"},
			},
		}, methodAll},
	}
	for _, test := range tests {
		assert.Equal(test.expected, MethodDef(test.typ))
	}
}

func TestInterfaceDef(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		typ      *GoType
		expected string
	}{
		{&GoType{Kind: reflect.String}, ""},
		{&GoType{
			Kind:        reflect.Interface,
			Name:        "geometry",
			PackagePath: "main",
			Methods: []*TypeMethod{
				{Name: "area", Type: &GoType{Kind: reflect.Func, FuncReturnVals: []*GoType{{Kind: reflect.Float64}}}},
				{Name: "perim", Type: &GoType{Kind: reflect.Func, FuncReturnVals: []*GoType{{Kind: reflect.Float64}}}},
			}}, ifDef},
		{&GoType{Kind: reflect.Interface, Name: "myEmptyIF", PackagePath: "main"}, "type myEmptyIF interface{}"},
	}
	for _, test := range tests {
		assert.Equal(test.expected, InterfaceDef(test.typ))
	}
}

const complexStructDef = `type myComplexStruct struct{
	myString string
	person *simpleStruct
	myArray [2]int
	mySlice []uint
	myChan chan struct{}
	myMap map[string]int
	myFunc func(string, int) uint
}`

const complexStructWithAnonDef = `type myComplexStruct struct{
	myString string
	person *simpleStruct
	myArray [2]int
	mySlice []uint
	myChan chan struct{}
	myMap map[string]int
	myFunc func(string, int) uint
	embeddedType
}`

const structWithFieldTag = "type myStruct struct{\n" +
	"	myString string	`json:\"String\"`\n" +
	"}"

const ifDef = `type geometry interface {
	area() float64
	perim() float64
}`

const methodAll = `func (myStruct) Read([]int8) (int, error)
func (myStruct) Close() error
func (myStruct) private()`
