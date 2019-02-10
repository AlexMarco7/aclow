package main

import (
	"net/url"

	"github.com/AlexMarco7/aclow"
)

func main() {

	startOpt := aclow.StartOptions{
		ModuleName:    "module_name",
		Host:          "localhost",
		Port:          4222,
		ClusterPort:   8222,
		ClusterRoutes: []*url.URL{
			//&url.URL{Host: fmt.Sprintf("localhost:%d", 8223)},
		},
	}

	var app = &aclow.App{}

	app.Start(startOpt)

	app.RegisterModule("module_name", []aclow.Node{
		&Ping{},
		&Pong{},
	})

	app.Wait()
}
