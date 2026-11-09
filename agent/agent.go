package agent

import (
	"context"

	"github.com/dslaskibert/ia04-projet-sma/types"
)

type Agent interface {
	ID() string
	Type() types.AgentType
	Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{})
}
