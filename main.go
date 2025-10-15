package main

import (
	"fmt"
	"os"

	"lem-in/internal/parser"
	"lem-in/internal/path"
	"lem-in/internal/scheduler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <input-file>")
		os.Exit(0)
	}
	res, err := parser.ParseFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// print original input (sanitized) first as required
	for _, ln := range res.OriginalLines {
		fmt.Println(ln)
	}
	fmt.Println()

	// find multiple disjoint shortest paths (no explicit maximum)
	paths := path.MultiPath(res.Graph, 0) // 0 => unlimited until none found

	if len(paths) == 0 {
		fmt.Println("ERROR: invalid data format, no path found")
		os.Exit(0)
	}

	// Run the scheduler that prints ant moves
	scheduler.Run(res.Ants, paths, res.Graph)
}
