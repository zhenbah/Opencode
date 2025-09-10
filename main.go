package main

import (
	"github.com/zhenbah/cryoncode/cmd"
	"github.com/zhenbah/cryoncode/internal/logging"
)

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	cmd.Execute()
}
