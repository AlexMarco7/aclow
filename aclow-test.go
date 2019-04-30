package aclow

import (
	"sync"
	"testing"
)

type Tester struct {
	App         *App
	address     string
	module      string
	assertFunc  func(Message, error)
	mocks       map[string]bool
	calledMocks map[string]bool
}

var lock = sync.RWMutex{}

func (t *Tester) Test(module string, node Node) {
	t.App = &App{}

	t.App.Start(StartOptions{
		Local: true,
	})

	t.address = node.Address()[0]
	t.module = module
	t.mocks = map[string]bool{}
	t.calledMocks = map[string]bool{}

	t.App.RegisterModule(module, []Node{node})
}

func (t *Tester) Mock(module string, address string, mock func(Message) (Message, error)) {
	lock.Lock()
	defer lock.Unlock()
	t.mocks[module+"@"+address] = true
	t.App.RegisterModule(module, []Node{&MockNode{
		Tester:        t,
		MockedModule:  module,
		MockedAddress: address,
		Mock:          mock,
	}})
}

func (t *Tester) Run(msg Message, testing *testing.T) {
	result, err := t.App.Call(t.module+"@"+t.address, msg)
	lock.RLock()
	defer lock.RUnlock()
	for k := range t.mocks {
		if !t.calledMocks[k] {
			testing.Errorf("%s wasn't called!", k)
		}
	}
	if t.assertFunc != nil {
		t.assertFunc(result, err)
	}
}

func (t *Tester) Assert(assertFunc func(Message, error)) {
	t.assertFunc = assertFunc
}

type MockNode struct {
	Tester        *Tester
	MockedModule  string
	MockedAddress string
	Mock          func(Message) (Message, error)
}

func (m *MockNode) Address() []string { return []string{m.MockedAddress} }

func (m *MockNode) Start(app *App) {}

func (m *MockNode) Execute(msg Message, call Caller) (Message, error) {
	lock.Lock()
	defer lock.Unlock()
	m.Tester.calledMocks[m.MockedModule+"@"+m.MockedAddress] = true
	return m.Mock(msg)
}
