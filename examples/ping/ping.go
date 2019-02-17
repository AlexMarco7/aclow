package main

import (
	"log"
	"time"

	"github.com/AlexMarco7/aclow"
)

type Ping struct{}

func (t *Ping) Address() []string { return []string{"ping"} }

func (t *Ping) Start(app *aclow.App) {
	time.Sleep(1 * time.Second)
	app.Publish("module_name@ping", aclow.Message{Body: int64(0)})
}

func (t *Ping) Execute(msg aclow.Message, call aclow.Caller) (aclow.Message, error) {
	count := msg.Body.(int64)

	log.Print("ping ", count)

	time.Sleep(1 * time.Second)
	if count >= 1000 {
		return msg, nil
	}
	return call("module_name@pong", aclow.Message{Body: count + 1})
}
