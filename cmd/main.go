package main

import (
	"fmt"

	cmd "github.com/JackalLabs/jackal-provider/cmd/jprovd"
)

func main() {
	rootCmd := cmd.NewRootCmd()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Error running root command!", err)
	}
}
