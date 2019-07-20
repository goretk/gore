package main

import "fmt"

type myComplexStruct struct {
	MyString string `json:"String"`
	person   *simpleStruct
	myArray  [2]int
	mySlice  []uint
	myChan   chan struct{}
	myMap    map[string]int
	myFunc   func(string, int) uint
	embeddedType
}

type simpleStruct struct {
	name string
	age  int
}

func (s *simpleStruct) String() string {
	return fmt.Sprintf("Name: %s | Age: %d", s.name, s.age)
}

type embeddedType struct {
	val int64
}

func main() {
	myPerson := &simpleStruct{name: "Test string", age: 42}
	complexStruct := &myComplexStruct{MyString: "A string", person: myPerson}
	fmt.Printf("Person: %v and a struct %v\n", myPerson, complexStruct)
}
