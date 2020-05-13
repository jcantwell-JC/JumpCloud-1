package main

import (
	JCServer "hash_pass/server"
	"log"
)

func main() {
	log.Printf("Starting server...")
	JCServer.HandleRequests()
	log.Printf("Service has shutdown")
}
