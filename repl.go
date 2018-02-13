package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
)

type repl struct {
	bufs    chan []byte
	close   chan struct{}
	release chan struct{}
}

// Read reads the bytes from the REPL into p. Returns the number of
// bytes read. There are no errors
func (r *repl) Read(p []byte) (int, error) {
	replBuf := <-r.bufs
	return copy(p, replBuf), nil
}

// Close closes the REPL
func (r *repl) Close() error {
	r.close <- struct{}{}
	return nil
}

// newRepl creates a new REPL
func newRepl(prompt string) io.ReadCloser {
	r := &repl{make(chan []byte), make(chan struct{}), make(chan struct{})}
	go r.run(prompt)
	return r
}

// run runs the REPL. Displays a line prompt designated by prompt.
// Blocks until closed.
func (r *repl) run(prompt string) {

	stdinread := func() chan []byte {
		c := make(chan []byte)
		go func() {
			stdin := bufio.NewReader(os.Stdin)
			msg, err := stdin.ReadString('\n')

			if err != nil {
				r.close <- struct{}{}
				close(c)
				return
			}

			msg = msg[:len(msg)-1] // remove newline...

			if runtime.GOOS == "windows" {
				// remove \r...
				msg = msg[:len(msg)-1]
			}

			c <- []byte(msg)
		}()
		return c
	}

	for {

		fmt.Print(prompt + " ")
		select {
		case input := <-stdinread():
			r.bufs <- input
		case <-r.close:
			return
		}
	}
}
