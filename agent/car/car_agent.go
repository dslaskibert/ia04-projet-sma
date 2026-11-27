package car


import (
	"context"
	"fmt"
	
	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/types"
)


var _ agent.Agent = (*Car)(nil)

func (c *Car) ID() types.CarID {
	return c.id
}

func (c *Car) Position() types.NodeID {
	return c.currentNode
}

func (c *Car) SetRegistry(reg types.Registry) {
	c.registry = reg
}

func (c *Car) Name() string {
	return fmt.Sprintf("car-%d", c.id)
}

func (c *Car) Type() types.AgentType {
	return types.AgentTypeCar
}

func (c *Car) Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tickIn:
			action := c.Step(now)
			fmt.Printf("[%s] tick %d received (pos: %d, dest: %d, strar: %s, state: %s, battery: %.1f%%, action: %s)\n",
				c.Name(), now, c.currentNode, c.destNode, c.strategy, c.state, c.batteryLevel*100, action.Type)

			doneOut <- struct{}{}
		}
	}
}