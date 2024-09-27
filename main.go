package main

import (
	"log"
	"os"

	"portForwarder/src/config"
	"portForwarder/src/parser"
	"portForwarder/src/portForwarder"
)

func main() {

	// open the file
	f, err := os.Open("ports.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Parse the file
	p := parser.NewKeyValueParser(f)
	result := p.Parse()

	/// Unmarshal the result into a portForwarder.Port
	ports := make([]portForwarder.Port, len(result))
	for i, r := range result {
		err = ports[i].UnmarshalMap(r)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Open config file
	fConf, err := os.Open("config.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer fConf.Close()

	// Parse the config file
	var configurations portForwarder.PortCredentials

	config.ParseConfig(fConf, &configurations)

	// show config
	log.Printf("Configurations: %+v", configurations)

	r := portForwarder.NewEdgeOsRouterPortForwarder(configurations)
	r.Connect()

	// remove all ports
	r.RemoveAllPorts()

	// Add the ports
	r.AddPorts(ports)
}
