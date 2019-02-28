package aclow

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

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
}

type Message struct {
	Header map[string]interface{}
	Body   interface{}
	Err    error
}

func (a *App) Start(opt StartOptions) {
	a.opt = opt
	a.Config = make(map[string]interface{})
	a.Resources = make(map[string]interface{})
	a.NodeMap = make(map[string]Node)

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
	a.Conn.Publish(address, msg)
}

func (a *App) Call(address string, d Message) (Message, error) {
	var r Message
	var err error
	localNode := a.NodeMap[address]
	if localNode == nil && !a.opt.Local {
		err = a.Conn.Request(address, d, &r, time.Second*30)
		if r.Err != nil {
			return Message{}, r.Err
		}
		return r, err
	} else {
		a.logIt("running ", address)
		return localNode.Execute(d, a.makeCaller(address))
	}
}

func (a *App) RegisterModule(moduleName string, nodes []Node) {
	for _, n := range nodes {
		go func(n Node) {
			for _, addr := range n.Address() {
				nodeAddress := moduleName + "@" + addr
				a.logIt("starting ", nodeAddress)

				a.NodeMap[nodeAddress] = n

				if !a.opt.Local {
					_, err := a.Conn.QueueSubscribe(nodeAddress, moduleName, func(_, reply string, msg Message) {
						a.logIt("running ", nodeAddress)

						go func(msg Message) {
							caller := a.makeCaller(nodeAddress)

							result, err := n.Execute(msg, caller)

							if err != nil {
								a.logIt(nodeAddress, " ", err.Error())
								if reply != "" {
									a.logIt(nodeAddress, " replying error")
									a.Conn.Publish(reply, Message{Err: err})
								}
							} else if reply != "" {
								a.logIt(nodeAddress, " replying success")
								a.Conn.Publish(reply, result)
							}
						}(msg)

					})

					if err != nil {
						println(err.Error)
					}
				}

			}

			n.Start(a)

		}(n)
	}
}

func (a *App) makeCaller(fromAddress string) Caller {
	return func(address string, m Message) (Message, error) {
		a.logIt(fromAddress, " calling ", address)
		return a.Call(address, m)
	}
}

func (a *App) logIt(values ...interface{}) {
	if a.opt.Debug {
		log.Print(values...)
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
		log.Fatal(err)
	}
}
