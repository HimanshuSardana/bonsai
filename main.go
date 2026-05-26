package main

import (
	"fmt"
	"os"

	"github.com/himanshusardana/bonsai.git/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: bonsai <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  init   Initialize a bonsai directory")
		fmt.Println("  scan   Scan for git repositories")
		fmt.Println("  serve  Start the HTTP server")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmd.Init()
	case "scan":
		path := "."
		if len(os.Args) > 2 {
			path = os.Args[2]
		}
		cmd.Scan(path)
	case "serve":
		port := "8000"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		cmd.Serve(port)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
