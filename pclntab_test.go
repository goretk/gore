package gore

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGo116PCLNTab(t *testing.T) {
	r := require.New(t)

	files := []string{
		"gold-linux-amd64-1.16.0",
		"gold-darwin-amd64-1.16.0",
		"gold-windows-amd64-1.16.0",
	}

	for _, gold := range files {
		t.Run(gold, func(t *testing.T) {
			win := filepath.Join("testdata", "gold", gold)

			f, err := Open(win)
			r.NoError(err)

			tab, err := f.PCLNTab()
			r.NoError(err)
			r.NotNil(tab)
		})
	}

}
