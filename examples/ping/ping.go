package main

import (
	aclow "aclow"
	"log"
	"time"
)

type Ping struct{}

func (t *Ping) Address() string { return "ping" }

func (t *Ping) Start(app *aclow.App) {
	time.Sleep(1 * time.Second)
	app.Publish("ping", aclow.Message{Body: int64(0)})
}

func (t *Ping) Execute(msg aclow.Message, call aclow.Caller) (aclow.Message, error) {
	count := msg.Body.(int64)

	log.Print("ping ", count)

	time.Sleep(1 * time.Second)
	if count >= 1000 {
		return msg, nil
	}
	return call("pong", aclow.Message{Body: count + 1})
}
