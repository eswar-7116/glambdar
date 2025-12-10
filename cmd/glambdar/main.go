package main

import (
	"log"

	"github.com/eswar-7116/glambdar/internal/api"
)

func main() {
	log.Println("Glambdar running on http://localhost:8080")
	if err := api.Router().Run(":8000"); err != nil {
		log.Fatal(err)
	}
}
