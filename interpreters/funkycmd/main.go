package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/faiface/funky"
	"github.com/faiface/funky/runtime"
)

func main() {
	program, cleanup := funky.Run("main")
	defer cleanup()
	in, out := bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout)
	defer out.Flush()
loop:
	for {
		switch program.Alternative() {
		case 0: // quit
			break loop
		case 1: // putc
			r := program.Field(0).Char()
			_, err := out.WriteRune(r)
			handleErr(err)
			if r == '\n' {
				out.Flush()
			}
			program = program.Field(1)
		case 2: // getc
			err := out.Flush()
			handleErr(err)
			r, _, err := in.ReadRune()
			if err == io.EOF {
				break loop
			}
			handleErr(err)
			program = program.Field(0).Apply(runtime.MkChar(r))
		}
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
