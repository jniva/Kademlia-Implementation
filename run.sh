#!/bin/bash
getent hosts kademlia_bootstrap_host | awk '{ print $1 }' > bootstrap_host
/usr/local/go/bin/go run main.go --bootstrap_ip $(cat bootstrap_host)
