package main

import (
	aclow "aclow"
	"net/url"
	"sync"
)

func main() {

	wg := &sync.WaitGroup{}
	wg.Add(1)

	startOpt := aclow.StartOptions{
		ModuleName:    "module",
		Host:          "localhost",
		Port:          4222,
		ClusterPort:   8222,
		ClusterRoutes: []*url.URL{
			//&url.URL{Host: fmt.Sprintf("localhost:%d", 8223)},
		},
	}

	var app = &aclow.App{}

	app.StartServer(startOpt)

	app.StartClient()

	app.RegisterModule("module_name", []aclow.Node{
		&Ping{},
		&Pong{},
	})

	wg.Wait()
}
