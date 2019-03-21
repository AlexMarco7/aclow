package aclow

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

type Logger struct {
	remoteWriter func(string)
}

type Log struct {
	logType          string // starting-execution|starting-call|receiving-call-response|ending-execution
	executionID      string
	executionAddress string
	address          string
	message          Message
	err              error
}

func (l *Logger) logIt(logMsg Log) {
	json, _ := json.Marshal(map[string]string{
		"log_type":          logMsg.logType,
		"execution_id":      logMsg.executionID,
		"execution_address": logMsg.executionAddress,
		"address":           logMsg.address,
		"message":           fmt.Sprintf("%#v", logMsg.message),
		"error":             fmt.Sprintf("%#v", logMsg.err),
	})
	log.Println("aclow:>>>" + string(json))
	//l.remoteWriter("aclow:>>>" + string(json))
}

func (l *Logger) start() {
	//l.remoteWriter = startLoggerServer()
}

func startLoggerServer() func(string) {
	port := 3333
	var l net.Listener
	var err error
	for {
		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			fmt.Println("Error starting logger server:", err.Error())
			port++
		} else {
			break
		}
	}
	connections := []net.Conn{}
	go func() {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Println("Error accepting logger server connection: ", err.Error())
				os.Exit(1)
			}
			connections = append(connections, conn)
		}
		for _, c := range connections {
			c.Close()
		}
	}()

	return func(log string) {
		for _, c := range connections {
			c.Write([]byte(log + "\n"))
		}
	}
}

func handleRequest(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	conn.Write([]byte("Message received."))
	conn.Close()
}
