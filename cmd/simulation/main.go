package main

import (
	"context"
	"fmt"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/agent/station"
	"github.com/dslaskibert/ia04-projet-sma/environment/gridmap"
	"github.com/dslaskibert/ia04-projet-sma/simulation"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

func main() {
	fmt.Println("Hello World")

	// Créer carte
	carte := gridmap.NewGridMap(10, 5, 100.0, [][2]int{{3, 3}, {7, 7}})
	carte.Print()

	// Créer simulation
	simu := simulation.NewSimu()

	// Créer agents
	agtTest := agent.NewAgentTest("test", types.AgentTypeTest)
	simu.RegisterAgent(agtTest)

	// Créer les stations (Judith) : une station sur chaque nœud station de la carte
	for i, node := range carte.Stations() {
		st := station.NewStation(types.StationID(i+1), node, 4, 50, 30)
		simu.RegisterAgent(st)
	}

	// Lancer la simulation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	simu.Run(ctx, 10)
}