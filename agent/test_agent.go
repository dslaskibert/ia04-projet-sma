package agent

import (
	"context"
	"fmt"

	"github.com/dslaskibert/ia04-projet-sma/types"
)

type TestAgent struct {
	name    string
	agtType types.AgentType
}

func NewAgentTest(name string, agtType types.AgentType) *TestAgent {
	return &TestAgent{name: name, agtType: agtType}
}

func (a *TestAgent) Name() string {
	return a.name
}

func (a *TestAgent) Type() types.AgentType {
	return a.agtType
}

func (a *TestAgent) Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tickIn:
			fmt.Printf("[%s] tick %d reçu\n", a.name, now)
			doneOut <- struct{}{}
		}
	}
}
