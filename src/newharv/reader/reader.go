package reader

import (
	"bufio"
)

type LineReader struct {
	buf *bufio.Reader
}

func NewLineReader(rd io.ReadCloser) LineReader {
	return LineReader{bufio.NewReader(rd)}
}

func (lr LineReader) Next(ctx context.Context) (line string, err error) {
	c = make(chan struct{})
	go func() {
		line, err = lr.next()
	}()

	select {
	case <-ctx.Done():
		lr.Close()
	case <-c:
		return line, err
	}
}

func (lr LineReader) next() (string, error) {
	skip := false
	for {
		line, isPrefix, err := lr.buf.ReadLine()
		if err != nil {
			return "", err
		}

		// skip to long line
		if isPrefix || skip {
			skip = isPrefix
			continue
		}

		return string(line), nil
	}
}
