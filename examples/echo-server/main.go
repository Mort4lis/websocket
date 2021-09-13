package main

import (
	"log"
)

func main() {
	app := NewApp()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
