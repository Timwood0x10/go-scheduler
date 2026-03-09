package main

import (
	"flag"
	"log"

	"algogpu/internal/server"
)

func main() {
	port := flag.String("port", ":50051", "gRPC server port")
	flag.Parse()

	log.Printf("Starting GPU Scheduler on %s", *port)
	server.Run(*port)
}
