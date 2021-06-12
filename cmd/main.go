package main

import (
	"github.com/Mort4lis/ws-echo-server/internal"
	"log"
)

func main() {
	app := internal.NewApp()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
