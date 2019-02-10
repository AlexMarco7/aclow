package main

import (
	"log"
	"time"

	"github.com/AlexMarco7/aclow"
)

type Pong struct{}

func (t *Pong) Address() string { return "pong" }

func (t *Pong) Start(app *aclow.App) {}

func (t *Pong) Execute(msg aclow.Message, call aclow.Caller) (aclow.Message, error) {
	count := msg.Body.(int64)

	log.Print("pong ", count)

	time.Sleep(1 * time.Second)
	if count >= 1000 {
		return msg, nil
	}

	return call("module_name@ping", aclow.Message{Body: count + 1})
}
