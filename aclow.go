package aclow

import (
	"fmt"
	"log"
	"net/url"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"

	server "github.com/nats-io/gnatsd/server"
	nats "github.com/nats-io/go-nats"
)

type StartOptions struct {
	Local         bool
	Debug         bool
	Host          string
	Port          int
	ClusterPort   int
	ClusterRoutes []*url.URL
}

type Caller = func(a string, d Message) (Message, error)

type Node interface {
	Address() []string
	Start(app *App)
	Execute(msg Message, call Caller) (Message, error)
}

type App struct {
	opt       StartOptions
	Conn      *nats.EncodedConn
	Config    map[string]interface{}
	Resources map[string]interface{}
	NodeMap   map[string]Node
	Logger    Logger
	onError   func(address string, err error)
}

type Message struct {
	Header map[string]interface{}
	Body   interface{}
}

type ReplyMessage struct {
	Message
	Err error
}

func (a *App) Start(opt StartOptions) {
	a.opt = opt
	a.Config = make(map[string]interface{})
	a.Resources = make(map[string]interface{})
	a.NodeMap = make(map[string]Node)

	a.Logger.start()

	if !opt.Local {
		a.startServer()
		a.startClient()
	}
}

func (a *App) Wait() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func (a *App) startServer() {
	server := server.New(&server.Options{
		Host:   a.opt.Host,
		Port:   a.opt.Port,
		Routes: a.opt.ClusterRoutes,
		Cluster: server.ClusterOpts{
			Port: a.opt.ClusterPort,
		},
	})

	go server.Start()

	time.Sleep(time.Second * 1)
}

func (a *App) startClient() {
	nc, err := nats.Connect(fmt.Sprintf("localhost:%d", a.opt.Port))
	a.Conn, _ = nats.NewEncodedConn(nc, nats.GOB_ENCODER)

	if err != nil {
		log.Fatal(err)
	}
}

func (a *App) Publish(address string, msg Message) {
	localNode := a.NodeMap[address]

	if localNode == nil && !a.opt.Local {
		a.Conn.Publish(address, msg)
	} else if localNode == nil {
		err := fmt.Errorf("Address '%s' not found!", address)
		log.Println(err)
	} else {
		var err error

		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered:", r)
				log.Println(string(debug.Stack()))
				err = fmt.Errorf("Error on call: %v", localNode.Address())
			}
		}()

		executionID := uuid.New().String()
		a.logIt(Log{
			logType:     "starting-execution",
			executionID: executionID,
			address:     address,
			message:     msg,
		})

		reply, err := localNode.Execute(msg, a.makeCaller(address, executionID))

		a.logIt(Log{
			logType:     "ending-execution",
			executionID: executionID,
			address:     address,
			message:     reply,
			err:         err,
		})

		if err != nil {
			log.Println("Error executing:", address)
			log.Println(string(debug.Stack()))
		}
	}
}

func (a *App) Call(address string, msg Message) (r Message, err error) {
	localNode := a.NodeMap[address]
	if localNode == nil && !a.opt.Local {
		replyMsg := ReplyMessage{}
		err = a.Conn.Request(address, msg, &replyMsg, time.Second*30)
		if replyMsg.Err != nil {
			return Message{}, replyMsg.Err
		}
		r.Body = replyMsg.Body
		r.Header = replyMsg.Header
		return r, err
	} else if localNode == nil {
		err := fmt.Errorf(fmt.Sprintf("Address '%s' not found!", address))
		if a.onError != nil {
			go func() { a.onError(address, err) }()
		}
		return Message{}, err
	} else {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered:", r)
				log.Println(string(debug.Stack()))
				err = fmt.Errorf("Error on call: %v", localNode.Address())
			}
		}()

		executionID := uuid.New().String()
		a.logIt(Log{
			logType:     "starting-execution",
			executionID: executionID,
			address:     address,
			message:     msg,
		})

		reply, err := localNode.Execute(msg, a.makeCaller(address, executionID))
		a.logIt(Log{
			logType:     "ending-execution",
			executionID: executionID,
			address:     address,
			message:     reply,
			err:         err,
		})

		if err != nil {
			log.Println("Error executing:", address, " => ", err.Error())
			log.Println(string(debug.Stack()))
			if a.onError != nil {
				go func() { a.onError(address, err) }()
			}
		}

		return reply, err
	}
}

func (a *App) RegisterModule(moduleName string, nodes []Node) {
	for _, n := range nodes {
		//go func(n Node) {
		for _, addr := range n.Address() {
			nodeAddress := moduleName + "@" + addr

			a.NodeMap[nodeAddress] = n

			if !a.opt.Local {
				_, err := a.Conn.QueueSubscribe(nodeAddress, moduleName, func(_, reply string, msg Message) {

					executionID := uuid.New().String()
					a.logIt(Log{
						logType:     "starting-execution",
						executionID: executionID,
						address:     nodeAddress,
						message:     msg,
					})

					go func(msg Message) {
						defer func() {
							if r := recover(); r != nil {
								log.Println("Recovered:", r)
								log.Println(string(debug.Stack()))
								a.Conn.Publish(reply, ReplyMessage{Err: fmt.Errorf("Error on call: %v", n.Address())})
							}
						}()

						caller := a.makeCaller(nodeAddress, executionID)

						result, err := n.Execute(msg, caller)

						a.logIt(Log{
							logType:     "ending-execution",
							executionID: executionID,
							address:     nodeAddress,
							message:     result,
							err:         err,
						})

						if err != nil {
							log.Println("Error executing:", nodeAddress)
							log.Println(string(debug.Stack()))
							if a.onError != nil {
								go func() { a.onError(nodeAddress, err) }()
							}

							if reply != "" {
								a.Conn.Publish(reply, ReplyMessage{Err: err})
							}
						} else if reply != "" {
							a.Conn.Publish(reply, result)
						}
					}(msg)

				})

				if err != nil {
					println(err.Error)
				}
			}

		}

		go n.Start(a)

		//}(n)
	}
}

func (a *App) makeCaller(fromAddress string, executionID string) Caller {
	return func(address string, m Message) (Message, error) {
		a.logIt(Log{
			logType:          "starting-call",
			executionID:      executionID,
			executionAddress: fromAddress,
			address:          address,
			message:          m,
		})
		reply, err := a.Call(address, m)
		//log.Println(err)
		a.logIt(Log{
			logType:          "receiving-call-response",
			executionID:      executionID,
			executionAddress: fromAddress,
			address:          address,
			message:          reply,
			err:              err,
		})
		return reply, err
	}
}

func (a *App) logIt(logMsg Log) {
	if a.opt.Debug {
		a.Logger.logIt(logMsg)
	}
}

func BodyAsString(r Message, err error) string {
	check(err)
	return r.Body.(string)
}

func BodyAsFloat64(r Message, err error) float64 {
	check(err)
	return r.Body.(float64)
}

func BodyAsInt64(r Message, err error) int64 {
	check(err)
	return r.Body.(int64)
}

func BodyAsBool(r Message, err error) bool {
	check(err)
	return r.Body.(bool)
}

func Body(r Message, err error) interface{} {
	check(err)
	return r.Body
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}

type Tuple []interface{}
