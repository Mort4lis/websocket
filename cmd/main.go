package main

import (
	"log"

	"github.com/Mort4lis/ws-echo-server/internal"
)

func main() {
	app := internal.NewApp()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
