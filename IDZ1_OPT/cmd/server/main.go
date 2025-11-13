package main

import (
	"log"
	"net/http"

	"idz1_opt/internal/server"
)

func main() {
	router := server.NewRouter()
	log.Println("Сервер запущен на http://localhost:8080")
	log.Println("Static files served from:", "static")
	log.Fatal(http.ListenAndServe(":8080", router))
}
