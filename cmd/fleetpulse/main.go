package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/haruto/fleetpulse/internal/api"
	"github.com/haruto/fleetpulse/internal/collector"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address for the FleetPulse API server")
	flag.Parse()

	service := collector.NewService()
	handler := api.NewHandler(service)

	log.Printf("fleetpulse listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
