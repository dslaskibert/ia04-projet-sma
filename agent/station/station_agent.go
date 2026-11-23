package station

import (
	"context"
	"fmt"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

// Vérifie que Station respecte bien l'interface Agent de David.
// S'il manque une méthode, le programme ne compile pas.
var _ agent.Agent = (*Station)(nil)

// Name : le nom de la station, par exemple "station-1"
func (s *Station) Name() string {
	return fmt.Sprintf("station-%d", s.id)
}

// Type : dit au moteur que cet agent est une station
func (s *Station) Type() types.AgentType {
	return types.AgentTypeStation
}

// Start : la "vie" de la station. Elle tourne dans sa propre goroutine.
// C'est la seule goroutine qui modifie la station, donc pas besoin de mutex.
// À chaque tour : elle lit les nouveaux messages, recharge pendant 1 minute,
// puis prévient le moteur qu'elle a fini.
func (s *Station) Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{}) {
	for {
		select {
		case <-ctx.Done(): // la simulation est finie : on s'arrête
			return
		case now := <-tickIn: // un nouveau tour commence
			s.receiveMessages(now)
			finished := s.step(now)
			for _, d := range finished {
				fmt.Printf("[%s] tour %d : voiture %d rechargée (%.1f kWh, %.0f centimes)\n",
					s.Name(), now, d.CarID, d.EnergyGiven, float64(d.Cost))
			}
			fmt.Printf("[%s] tour %d : %d/%d bornes libres, %d en file\n",
				s.Name(), now, s.freePumps(), len(s.pumps), len(s.queue))
			doneOut <- struct{}{} // "j'ai fini mon tour"
		}
	}
}
