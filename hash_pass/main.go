package main

import (
	"fmt"
	JCServer "hash_pass/server"
	"log"
	"os"
	"strconv"
	"syscall"
)

const (
	minPort = 1024
	maxPort = 65535
)
func main() {
	listenPort := JCServer.ListenPort

	if len(os.Args) > 1{
		port, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Printf("Invalid port value '%s'\n",os.Args[1])
			syscall.Exit(-1)
		}
		if port <= minPort || port > maxPort {
			fmt.Printf("Port must be in range of 1024 < port < 65536\n")
			syscall.Exit(-1)
		}
		listenPort = port
	}

	log.Printf("Starting server on port %d",listenPort)
	JCServer.StartServer(listenPort)
	log.Printf("Service has shutdown")
}
