package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/agent/car"
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

	orig, dest := carte.RandomOriginDestination(rand.New(rand.NewSource(1)))
	
	// Créer agents
	agtTest := agent.NewAgentTest("test", types.AgentTypeTest)
	simu.RegisterAgent(agtTest)
	
	car := car.NewCar(1, orig, dest, 0.8, 100, 0.2, carte, rand.New(rand.NewSource(1)))
	simu.RegisterAgent(car)

	// Lancer la simulation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	simu.Run(ctx, 10)

}