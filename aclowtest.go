package aclow

type Tester struct {
	app        *App
	address    string
	assertFunc func(Message, error)
}

func (t *Tester) Test(module string, node Node) {
	t.app = &App{}

	t.app.Start(StartOptions{
		Local: true,
	})

	t.address = node.Address()[0]

	t.app.RegisterModule(module, []Node{node})
}

func (t *Tester) Mock(module string, address string, mock func(Message) (Message, error)) {
	t.app.RegisterModule(module, []Node{&MockNode{
		MockedAddress: address,
		Mock:          mock,
	}})
}

func (t *Tester) Run(msg Message) {
	result, err := t.app.Call(t.address, msg)
	if t.assertFunc != nil {
		t.assertFunc(result, err)
	}
}

func (t *Tester) Assert(assertFunc func(Message, error)) {
	t.assertFunc = assertFunc
}

type MockNode struct {
	MockedAddress string
	Mock          func(Message) (Message, error)
}

func (t *MockNode) Address() []string { return []string{t.MockedAddress} }

func (t *MockNode) Start(app *App) {}

func (t *MockNode) Execute(msg Message, call Caller) (Message, error) {
	return t.Mock(msg)
}
