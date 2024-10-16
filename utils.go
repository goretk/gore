package gore

import "io"

func tryClose(r io.ReaderAt) error {
	if c, ok := r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
