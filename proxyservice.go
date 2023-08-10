package main

import (
	"flag"
	"github.com/Rorical/clip-as-service-proxy/proxyservice"
	log "github.com/sirupsen/logrus"
)

func main() {
	var configPath string
	var addr string
	var debug bool
	flag.StringVar(&configPath, "config", "config.yaml", "Config File Path")
	flag.StringVar(&addr, "address", ":50051", "Service Listen Address")
	flag.BoolVar(&debug, "debug", false, "Debug Log Level")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	panic(proxyservice.Serve(addr, configPath))
}
