package cmd

import (
	"fmt"
	"os"
)

const bonsaiYaml = `header: Bonsai
description: A lightweight Git frontend
`

func Init() {
	if _, err := os.Stat("bonsai.yaml"); err == nil {
		fmt.Println("bonsai.yaml already exists")
		return
	}

	if err := os.WriteFile("bonsai.yaml", []byte(bonsaiYaml), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bonsai.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Initialized empty bonsai project")
}
