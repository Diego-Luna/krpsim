package main

import (
	"fmt"
	"io"
	"os"

	"krpsim/internal/config"
	"krpsim/internal/report"
	"krpsim/internal/verify"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: krpsim_verif <file> <result_to_test>")
		os.Exit(1)
	}

	cfg, err := config.ParseFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse config %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	var in io.Reader = os.Stdin
	if os.Args[2] != "-" {
		f, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: open trace %q: %v\n", os.Args[2], err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}

	res, err := verify.Replay(cfg, in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read trace: %v\n", err)
		os.Exit(1)
	}
	report.Verify(os.Stdout, cfg, res)
	if !res.OK {
		os.Exit(1)
	}
}
