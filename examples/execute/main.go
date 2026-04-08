package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SpringMT/memsh"
)

func main() {
	mgr := memsh.NewManager()
	s := mgr.Open()
	if err := s.Load([]memsh.File{
		{Path: "/input/app.log", Content: []byte("INFO boot\nERROR b\nERROR a\nERROR a\n")},
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer mgr.Close(s.ID())

	result, err := s.Execute(context.Background(), `grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("output path: %s\n", result.OutputPath)
	fmt.Print(string(result.Output))
}
