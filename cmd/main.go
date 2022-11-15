package main

import (
	cmd "github.com/JackalLabs/jackal-provider/cmd/jprovd"
)

func main() {
	root := cmd.NewRootCmd()

	root.Execute()
}
