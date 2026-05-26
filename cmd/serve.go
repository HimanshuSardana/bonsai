package cmd

import (
	"fmt"
	"net/http"
	"os"
)

func Serve(port string) {
	addr := "0.0.0.0:" + port
	fmt.Printf("Serving bonsai on %s\n", addr)

	handler := http.FileServer(http.Dir(".bonsai"))
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
