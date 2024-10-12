package gore

import (
	"errors"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// Symbol A generic representation of [debug/elf.Symbol], [debug/pe.Symbol], and [debug/macho.Symbol].
type Symbol struct {
	// Name of the symbol.
	Name string
	// Value of the symbol.
	Value uint64
	// Size of the symbol. Only accurate on ELF files. For Mach-O and PE files, it was inferred by looking at the next symbol.
	Size uint64
}
