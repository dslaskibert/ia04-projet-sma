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
	carte := gridmap.NewGridMap(
		cfg.Map.Width,
		cfg.Map.Height,
		cfg.Map.NodeSpacingM,
		cfg.Map.StationPositions,
	)
	carte.Print()

	// Simulation
	simu := simulation.NewSimu()

	// Agent de test
	agtTest := agent.NewAgentTest("test", types.AgentTypeTest)
	simu.RegisterAgent(agtTest)

	// Voitures : chacune a son propre RNG, seedé depuis le RNG principal
	// TODO: passer aussi cfg.Cars.BatteryThreshold à NewCar quand le champ sera ajouté
	for i := 1; i <= cfg.Cars.Count; i++ {
		orig, dest := carte.RandomOriginDestination(rng)
		carRng := rand.New(rand.NewSource(rng.Int63()))
		c := car.NewCar(
			types.CarID(i),
			orig, dest,
			cfg.Cars.InitialBattery,
			cfg.Cars.MaxBatteryKWh,
			cfg.Cars.ConsumptionKWhPerKm,
			carte,
			carRng,
		)
		simu.RegisterAgent(c)
	}

	// Stations
	// TODO: quand la PR euros de Judith sera mergée, types.Price(cfg.Stations.PriceEurosPerKWh) sans le *100
	for i, node := range carte.Stations() {
		st := station.NewStation(
			types.StationID(i+1),
			node,
			cfg.Stations.PumpsPerStation,
			cfg.Stations.PumpPowerKW,
			types.Price(cfg.Stations.PriceEurosPerKWh*100),
		)
		simu.RegisterAgent(st)
	}

	// Lancement
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	simu.Run(ctx, cfg.MaxTicks)
}
