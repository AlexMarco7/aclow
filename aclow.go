package aclow

import (
	"fmt"
	"log"
	"net/url"
	"runtime"
	"sync"
	"time"

	server "github.com/nats-io/gnatsd/server"
	nats "github.com/nats-io/go-nats"
)

type StartOptions struct {
	ModuleName    string
	Debug         bool
	Host          string
	Port          int
	ClusterPort   int
	ClusterRoutes []*url.URL
}

type Caller = func(a string, d Message) (Message, error)

type Node interface {
	Address() string
	Start(app *App)
	Execute(msg Message, call Caller) (Message, error)
}

type App struct {
	opt       StartOptions
	conn      *nats.EncodedConn
	config    map[string]*interface{}
	resources map[string]interface{}
}

type Message struct {
	Header map[string]interface{}
	Body   interface{}
	Err    error
}

func (a *App) Start(opt StartOptions) {
	a.opt = opt
	a.startServer()
	a.startClient()
}

func (a *App) Wait() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func (a *App) startServer() {
	server := a.createServer(a.opt)

	go server.Start()

	time.Sleep(time.Second * 1)
}

func (a *App) startClient() {
	nc, err := nats.Connect(fmt.Sprintf("localhost:%d", a.opt.Port))
	a.conn, _ = nats.NewEncodedConn(nc, nats.GOB_ENCODER)

	if err != nil {
		log.Fatal(err)
	}
}

func (a *App) Publish(address string, msg Message) {
	a.conn.Publish(address, msg)
}

func (a *App) RegisterModule(moduleName string, nodes []Node) {
	for i := 0; i < runtime.NumCPU(); i++ {
		go a.subscribeAll(moduleName, nodes)
	}
}

func (a *App) subscribeAll(moduleName string, nodes []Node) {
	for i, n := range nodes {
		go func(nodeIndex int, n Node) {
			nodeAddress := moduleName + "@" + n.Address()
			a.logIt("starting ", nodeAddress, " ", nodeIndex)
			n.Start(a)
			_, err := a.conn.QueueSubscribe(nodeAddress, moduleName, func(_, reply string, msg Message) {
				a.logIt("running ", nodeAddress, " ", nodeIndex)

				go func(nodeAddress string, nodeIndex int, msg Message) {
					caller := func(address string, d Message) (Message, error) {
						a.logIt(nodeAddress, " ", nodeIndex, " calling ", address)
						var r Message
						err := a.conn.Request(address, d, &r, time.Second*30)
						if r.Err != nil {
							return Message{}, r.Err
						}
						return r, err
					}

					result, err := n.Execute(msg, caller)

					if err != nil {
						a.logIt(nodeAddress, " ", nodeIndex, " ", err.Error())
						if reply != "" {
							a.logIt(nodeAddress, " ", nodeIndex, " replying error")
							a.conn.Publish(reply, Message{Err: err})
						}
					} else if reply != "" {
						a.logIt(nodeAddress, " ", nodeIndex, " replying success")
						a.conn.Publish(reply, result)
					}
				}(nodeAddress, nodeIndex, msg)

			})
			if err != nil {
				println(err.Error)
			}
		}(i, n)
	}
}

func (a *App) logIt(values ...interface{}) {
	if a.opt.Debug {
		log.Print(values...)
	}
}

func (a *App) createServer(opt StartOptions) *server.Server {

	s := server.New(&server.Options{
		Host:   opt.Host,
		Port:   opt.Port,
		Routes: opt.ClusterRoutes,
		Cluster: server.ClusterOpts{
			Port: opt.ClusterPort,
		},
	})

	return s

}
