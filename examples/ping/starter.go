package main

import (
	"net/url"
	"time"

	"github.com/AlexMarco7/aclow"
)

func main() {

	startOpt := aclow.StartOptions{
		Debug:         true,
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

	time.Sleep(1 * time.Second)
	app.Publish("module_name@ping", aclow.Message{Body: int64(0)})

	app.Wait()
}
