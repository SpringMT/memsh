package main

import (
	"fmt"
	"os"

	"github.com/SpringMT/memsh/memfs"
)

func main() {
	fs := memfs.New()

	file, err := fs.Write("/input/app.log", []byte("INFO boot\nERROR failed\n"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	data, stat, err := fs.Read("/input/app.log")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("written path=%s size=%d\n", file.Path, file.Size)
	fmt.Printf("read path=%s size=%d\n", stat.Path, stat.Size)
	fmt.Println(string(data))
}
