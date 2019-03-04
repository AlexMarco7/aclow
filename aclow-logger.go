package aclow

import (
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
	log.Println(logMsg)
	l.remoteWriter(fmt.Sprintf("%#v", logMsg))
}

func (l *Logger) start() {
	l.remoteWriter = startLoggerServer()
}

func startLoggerServer() func(string) {
	l, err := net.Listen("tcp", "localhost:3333")
	if err != nil {
		fmt.Println("Error starting logger server:", err.Error())
		os.Exit(1)
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
