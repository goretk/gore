// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testParseStructType(t *testing.T) {
	// assert := assert.New(t)
	// data := []byte{
	// 	0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x70, 0x5b, 0x3c, 0x1a, 0x07, 0x08, 0x08, 0x19, 0xd0, 0xae, 0x48, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0xcc, 0x79, 0x4c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79, 0x6e, 0x00, 0x00, 0xc0, 0xaa, 0x00, 0x00,
	// 	0xf8, 0x73, 0x48, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc0, 0x1f, 0x4a, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0xf8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x99, 0x76, 0x48, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x67, 0x49, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xee, 0x71, 0x48, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x80, 0x60, 0x49, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// }

	// // val := typeParse(context.TODO(), data)
	// val := typeParse(context.TODO(), 0, data, 0)

	// assert.Equal(reflect.Struct, val.kind, "Wrong kind parsed")
	// assert.Equal(int64(0x18), val.length, "Wrong length parsed")
	// assert.Equal(int32(0x6e79), val.nameOff, "Wrong nameOffset parsed")
}

func TestGetTypes(t *testing.T) {
	goldFiles, err := getGoldenResources()
	if err != nil || len(goldFiles) == 0 {
		// Golden folder does not exist
		t.Skip("No golden files")
	}
	for _, test := range goldFiles {
		t.Run("get_types_"+test, func(t *testing.T) {
			t.Parallel()
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
			assert.NoError(err, "Should parse with no error")

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
			FuncArgs:       []*GoType{&GoType{Kind: reflect.String}, &GoType{Kind: reflect.Int}},
			FuncReturnVals: []*GoType{&GoType{Kind: reflect.Uint}},
		}, "func(string, int) uint"},
		{&GoType{
			Kind:           reflect.Func,
			FuncArgs:       []*GoType{&GoType{Kind: reflect.String}, &GoType{Kind: reflect.Int}},
			FuncReturnVals: []*GoType{&GoType{Kind: reflect.Uint}, &GoType{Kind: reflect.Struct}},
		}, "func(string, int) (uint, struct{})"},
		{&GoType{
			Kind:     reflect.Func,
			FuncArgs: []*GoType{&GoType{Kind: reflect.String}},
		}, "func(string)"},
		{&GoType{
			Kind:           reflect.Func,
			FuncReturnVals: []*GoType{&GoType{Kind: reflect.Uint}},
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
				&GoType{FieldName: "myString", Kind: reflect.String},
				&GoType{FieldName: "person", Kind: reflect.Ptr, Element: &GoType{Kind: reflect.Struct, Name: "simpleStruct"}},
				&GoType{FieldName: "myArray", Kind: reflect.Array, Length: 2, Element: &GoType{Kind: reflect.Int}},
				&GoType{FieldName: "mySlice", Kind: reflect.Slice, Element: &GoType{Kind: reflect.Uint}},
				&GoType{FieldName: "myChan", Kind: reflect.Chan, Element: &GoType{Kind: reflect.Struct}},
				&GoType{FieldName: "myMap", Kind: reflect.Map, Element: &GoType{Kind: reflect.Int}, Key: &GoType{Kind: reflect.String}},
				&GoType{FieldName: "myFunc", Kind: reflect.Func, FuncArgs: []*GoType{&GoType{Kind: reflect.String}, &GoType{Kind: reflect.Int}}, FuncReturnVals: []*GoType{&GoType{Kind: reflect.Uint}}},
			}}, complexStructDef},
		{&GoType{
			Kind: reflect.Struct,
			Name: "myComplexStruct",
			Fields: []*GoType{
				&GoType{FieldName: "myString", Kind: reflect.String},
				&GoType{FieldName: "person", Kind: reflect.Ptr, Element: &GoType{Kind: reflect.Struct, Name: "simpleStruct"}},
				&GoType{FieldName: "myArray", Kind: reflect.Array, Length: 2, Element: &GoType{Kind: reflect.Int}},
				&GoType{FieldName: "mySlice", Kind: reflect.Slice, Element: &GoType{Kind: reflect.Uint}},
				&GoType{FieldName: "myChan", Kind: reflect.Chan, Element: &GoType{Kind: reflect.Struct}},
				&GoType{FieldName: "myMap", Kind: reflect.Map, Element: &GoType{Kind: reflect.Int}, Key: &GoType{Kind: reflect.String}},
				&GoType{FieldName: "myFunc", Kind: reflect.Func, FuncArgs: []*GoType{&GoType{Kind: reflect.String}, &GoType{Kind: reflect.Int}}, FuncReturnVals: []*GoType{&GoType{Kind: reflect.Uint}}},
				&GoType{FieldAnon: true, Kind: reflect.Struct, Name: "embeddedType"},
			}}, complexStructWithAnonDef},
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
				&TypeMethod{Name: "Read", Type: &GoType{
					Kind:           reflect.Func,
					FuncArgs:       []*GoType{&GoType{Kind: reflect.Slice, Element: &GoType{Kind: reflect.Int8}}},
					FuncReturnVals: []*GoType{&GoType{Kind: reflect.Int}, &GoType{Kind: reflect.Interface, Name: "error"}}}},
				&TypeMethod{Name: "Close", Type: &GoType{
					Kind:           reflect.Func,
					FuncReturnVals: []*GoType{&GoType{Kind: reflect.Interface, Name: "error"}}}},
				&TypeMethod{Name: "private"},
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
				&TypeMethod{Name: "area", Type: &GoType{Kind: reflect.Func, FuncReturnVals: []*GoType{&GoType{Kind: reflect.Float64}}}},
				&TypeMethod{Name: "perim", Type: &GoType{Kind: reflect.Func, FuncReturnVals: []*GoType{&GoType{Kind: reflect.Float64}}}},
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

const ifDef = `type geometry interface {
	area() float64
	perim() float64
}`

const methodAll = `func (myStruct) Read([]int8) (int, error)
func (myStruct) Close() error
func (myStruct) private()`
