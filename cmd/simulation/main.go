package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/agent/car"
	"github.com/dslaskibert/ia04-projet-sma/agent/station"
	"github.com/dslaskibert/ia04-projet-sma/config"
	"github.com/dslaskibert/ia04-projet-sma/environment/gridmap"
	"github.com/dslaskibert/ia04-projet-sma/simulation"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

func main() {
	// Chargement de la config
	cfg, err := config.Load("configs/minimal.yaml")
	if err != nil {
		log.Fatalf("Chargement config: %v", err)
	}
	fmt.Printf("Config chargée: %+v\n", cfg)

	// RNG principal seedé (reproductibilité)
	rng := rand.New(rand.NewSource(cfg.Seed))

	// Carte
	// TODO: passer stationPositions dans la config
	stationPositions := [][2]int{{3, 2}, {7, 3}}
	carte := gridmap.NewGridMap(cfg.Map.Width, cfg.Map.Height, 100.0, stationPositions)
	carte.Print()

	// Simulation
	simu := simulation.NewSimu()

	// Agent de test
	agtTest := agent.NewAgentTest("test", types.AgentTypeTest)
	simu.RegisterAgent(agtTest)

	// Voitures : chacune a son propre RNG, seedé depuis le RNG principal
	for i := 1; i <= cfg.Cars.Count; i++ {
		orig, dest := carte.RandomOriginDestination(rng)
		carRng := rand.New(rand.NewSource(rng.Int63()))
		c := car.NewCar(
			types.CarID(i),
			orig, dest,
			cfg.Cars.InitialBattery,
			cfg.Cars.MaxBattery,
			cfg.Cars.Consumption,
			carte,
			carRng,
		)
		simu.RegisterAgent(c)
	}

	// Stations
	// TODO: ajouter pump_power_kw et price_centimes_per_kwh dans la config
	for i, node := range carte.Stations() {
		st := station.NewStation(
			types.StationID(i+1),
			node,
			cfg.Stations.PumpsPerStation,
			50,
			30,
		)
		simu.RegisterAgent(st)
	}

	// Lancement
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	simu.Run(ctx, cfg.MaxTicks)
}
