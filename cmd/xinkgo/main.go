// Command xinkgo is the entry point of the Xinkgo desktop application.
//
// It only hands control to internal/app, which owns the application's
// lifetime, so that the entry point stays free of logic and testable
// packages stay usable from tests.
//
//	go run ./cmd/xinkgo        # run the app
//	go tool mygo dev           # run it with live reload
package main

import (
	"log"

	"github.com/helightxu/xinkgo/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
