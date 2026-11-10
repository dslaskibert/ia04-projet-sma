package simulation

import (
	"context"
	"fmt"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

type AgentStruct struct {
	agent   agent.Agent
	tickIn  chan types.SimTime
	doneOut chan struct{}
}

type Simu struct {
	agents []AgentStruct
}

func (simu *Simu) CreateAgent(a agent.Agent) {
	newAgent := AgentStruct{agent: a, tickIn: make(chan types.SimTime), doneOut: make(chan struct{})}
	simu.agents = append(simu.agents, newAgent)
}

func (simu *Simu) Run(ctx context.Context, maxTicks int) {
	// Création des agents
	fmt.Println("Log: Création des agents")
	for _, agt := range simu.agents {
		go agt.agent.Start(ctx, agt.tickIn, agt.doneOut)
	}

	// Boucle de simulation
	fmt.Println("Log: Démarrage de la boucle principale de la simulation")
	for tick := 1; tick < maxTicks; tick++ {
		fmt.Printf("Début du tick %d\n", tick)
		now := types.SimTime(tick)

		// envoi du tick
		for _, agt := range simu.agents {
			agt.tickIn <- now
		}

		// attente de la réponse de tout le monde
		for _, agt := range simu.agents {
			<-agt.doneOut
		}

		fmt.Printf("Fin du tick %d\n", tick)
	}
}
