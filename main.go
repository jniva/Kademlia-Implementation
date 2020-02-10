package main

import (
	"flag"
	"fmt"
	"os"

	"kademlia"
)

const defaultPort = "8000"

func main() {
	// parse arguments
	var port = flag.String("port", defaultPort, "specify port for the connections.")
	var bootstrapIP = flag.String("bootstrap_ip", "kademlia_bootstrap_host", "The bootstrap node IP address to join the network.")
	var bootstrapPort = flag.String("bootstrap_port", defaultPort, "The port of bootstrap node")
	flag.Parse()

	fmt.Println("Bootstrap Address: " + *bootstrapIP + ":" + *bootstrapPort)
	fmt.Println("Node is listeneing to Port: " + *port)

	network := kademlia.NewNetwork(*port, *bootstrapIP, *bootstrapPort)

	go network.Listen(*port)
	//go network.Listen(network.Contact)
	network.CliHelper(os.Stdin)

}
