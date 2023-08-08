package main

import (
	"flag"
	"github.com/Rorical/clip-as-service-proxy/proxyservice"
)

func main() {
	var configPath string
	var addr string
	flag.StringVar(&configPath, "config", "config.yaml", "Config File Path")
	flag.StringVar(&addr, "address", ":50051", "Service Listen Address")
	flag.Parse()

	panic(proxyservice.Serve(addr, configPath))
}
