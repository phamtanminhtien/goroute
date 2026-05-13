package main

import (
	"log"

	"github.com/phamtanminhtien/goroute/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
