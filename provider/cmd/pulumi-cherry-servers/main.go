package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/caliban0/pulumi-cherry-servers/provider"
)

func main() {
	provider, err := p.Provider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}

	err = provider.Run(context.Background(), p.Name, p.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}
