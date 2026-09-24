package main

import (
	"os"

	"github.com/metruzanca/mask/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
