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
// À chaque tour : elle reçoit le numéro du tour, l'affiche,
// puis prévient le moteur qu'elle a fini.
func (s *Station) Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{}) {
	for {
		select {
		case <-ctx.Done(): // la simulation est finie : on s'arrête
			return
		case now := <-tickIn: // un nouveau tour commence
			fmt.Printf("[%s] tour %d reçu\n", s.Name(), now)
			// plus tard (jalon 2) : s.step(now) pour recharger les voitures
			doneOut <- struct{}{} // "j'ai fini mon tour"
		}
	}
}
