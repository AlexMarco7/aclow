package main

import (
	"fmt"
	aclow "go_pocs/aclow"
	"time"
)

type Pong struct{}

func (t *Pong) Address() string { return "pong" }

func (t *Pong) Start(app *aclow.App) {}

func (t *Pong) Execute(msg aclow.Message, call aclow.Caller) (aclow.Message, error) {
	count := msg.Body.(int64)

	fmt.Print("pong ", count)

	time.Sleep(1 * time.Second)
	if count >= 1000 {
		return msg, nil
	}

	return call("ping", aclow.Message{Body: count + 1})
}
