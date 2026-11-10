package main

import (
	"log"
	"os"

	"github.com/ayn2op/discordo/cmd"
	"github.com/ayn2op/discordo/internal/cli"
)

func main() {
	if err := cmd.Run(); err != nil {
		if code, ok := cli.ExitCode(err); ok {
			os.Exit(code)
		}

		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
