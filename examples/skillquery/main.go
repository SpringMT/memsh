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
	defer mgr.Close(s.ID())

	if err := s.Load([]memsh.File{
		{
			Path: "/input/data.json",
			Content: []byte(`{
  "entries": [
    {"map": "AI Product Proposal", "label": "Need clearer ICP", "owner": "PM"},
    {"map": "AI Product Proposal", "label": "Missing pricing narrative", "owner": "BizDev"},
    {"map": "Internal Ops", "label": "Manual approvals remain", "owner": "Ops"}
  ]
}`),
		},
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result, err := s.Execute(
		context.Background(),
		`json.query '.entries[] | select(.map == "AI Product Proposal")' /input/data.json | json.to_text > /output/map.txt`,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exit=%d stderr=%s\n", result.ExitCode, string(result.Stderr))
		os.Exit(1)
	}

	fmt.Printf("output path: %s\n", result.OutputPath)
	fmt.Print(string(result.Output))
}
