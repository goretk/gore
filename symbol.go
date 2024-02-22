package gore

import (
	"errors"
	"sync"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// symbol A primitive representation of a symbol.
type symbol struct {
	Name  string
	Value uint64
	Size  uint64
}

type symbolTableOnce struct {
	*sync.Once
	table map[string]symbol
	err   error
}

func newSymbolTableOnce() *symbolTableOnce {
	return &symbolTableOnce{Once: &sync.Once{}, table: make(map[string]symbol)}
}
