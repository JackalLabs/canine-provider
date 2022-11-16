package main

import (
	"fmt"
)

func main() {
	rootCmd := NewRootCmd()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Error running root command!", err)
	}
}
