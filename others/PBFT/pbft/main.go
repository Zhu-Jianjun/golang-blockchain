package main

import (
	"PBFT/pbft/network" //where to invoke
	"os"                //standard libiary
)

func main() {
	nodeID := os.Args[1]                //https://gobyexample.com/command-line-arguments, check this link. e.g, On the terminal, main.exe Apple, Apple will be catched as Args[1]
	server := network.NewServer(nodeID) // to invoke the function NewServer from proxy_server.go

	server.Start()
}

/*
Basically, this main program is to start the servers.
*/
