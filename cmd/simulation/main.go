package main

import (
	"context"
	"fmt"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/agent/station"
	"github.com/dslaskibert/ia04-projet-sma/simulation"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

func main() {
	fmt.Println("Hello World")

	// Créer carte
	// carte := gridmap.NewGridMap(10, 10, 100.0, [][2]int{{3, 3}, {7, 7}})

	// Créer simulation
	simu := simulation.NewSimu()

	// Créer agents
	agtTest := agent.NewAgentTest("test", types.AgentTypeTest)
	simu.RegisterAgent(agtTest)

	// Créer les stations (Judith)
	station1 := station.NewStation(1, 33, 4, 50, 30)
	station2 := station.NewStation(2, 77, 4, 50, 30)
	simu.RegisterAgent(station1)
	simu.RegisterAgent(station2)

	// Lancer la simulation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	simu.Run(ctx, 10)
}
