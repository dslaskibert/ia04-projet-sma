package simulation

import (
	"context"
	"fmt"
	"sort"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

type agentSlot struct {
	agent   agent.Agent
	tickIn  chan types.SimTime
	doneOut chan struct{}
}

// moteur de la simulation
// - gère les ticks de simulation
// - liste des agents
type Simu struct {
	agents []agentSlot

	// Registre des agents
	stationByID map[types.StationID]types.StationHandle
	carByID     map[types.CarID]types.CarHandle
}

// Vérification à la compilation que *Simu implémente types.Registry.
var _ types.Registry = (*Simu)(nil)

func NewSimu() *Simu {
	return &Simu{
		stationByID: make(map[types.StationID]types.StationHandle),
		carByID:     make(map[types.CarID]types.CarHandle),
	}
}

// Enregistre un agent dans la simulation
// Si l'agent implémente StationHandle ou CarHandle il est indexé dans le registre
func (simu *Simu) RegisterAgent(a agent.Agent) {
	newAgent := agentSlot{
		agent:   a,
		tickIn:  make(chan types.SimTime),
		doneOut: make(chan struct{}),
	}
	simu.agents = append(simu.agents, newAgent)

	// Indexation dans le registre par type assertion
	// Panic volontaire sur ID dupliqué
	if sh, ok := a.(types.StationHandle); ok {
		if _, exists := simu.stationByID[sh.ID()]; exists {
			panic(fmt.Sprintf("RegisterAgent: station ID %d déjà enregistrée", sh.ID()))
		}
		simu.stationByID[sh.ID()] = sh
	}
	if ch, ok := a.(types.CarHandle); ok {
		if _, exists := simu.carByID[ch.ID()]; exists {
			panic(fmt.Sprintf("RegisterAgent: voiture ID %d déjà enregistrée", ch.ID()))
		}
		simu.carByID[ch.ID()] = ch
	}
}

// Registre
func (simu *Simu) StationByID(id types.StationID) types.StationHandle {
	return simu.stationByID[id] // renvoie nil si absent, comportement Go idiomatique
}

func (simu *Simu) AllStations() []types.StationHandle {
	out := make([]types.StationHandle, 0, len(simu.stationByID))
	for _, s := range simu.stationByID {
		out = append(out, s)
	}
	// Tri par ID pour garantir la reproductibilité car les map Go sont non déterministes
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

func (simu *Simu) CarByID(id types.CarID) types.CarHandle {
	return simu.carByID[id]
}

func (simu *Simu) Run(ctx context.Context, maxTicks int) {
	// Création des agents
	fmt.Println("Log: Création des agents")
	for _, agt := range simu.agents {
		go agt.agent.Start(ctx, agt.tickIn, agt.doneOut)
	}

	// Boucle de simulation
	fmt.Println("Log: Démarrage de la boucle principale de la simulation")
	for tick := 1; tick <= maxTicks; tick++ {
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
