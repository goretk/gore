package gore

import (
	"errors"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// Symbol A primitive representation of a symbol.
type Symbol struct {
	Name  string // Name of the symbol.
	Value uint64 // Value of the symbol.
	// Size of the symbol. Only accurate on ELF files. For Mach-O and PE files, it was inferred by looking at the next symbol.
	Size uint64
}
