package station

import (
	"sync"

	"github.com/dslaskibert/ia04-projet-sma/types"
)

// ChargeRequest
type ChargeRequest struct {
	CarID     types.CarID
	NeededKWh float64 // énergie qu'il reste à lui donner
}

// une borne de recharge de la station
type Pump struct {
	PowerKW float64        // puissance de la borne (en kW)
	Broken  bool           // true si la borne est en panne
	Car     *ChargeRequest // voiture branchée (nil = borne libre)
}

// Station : l'agent station
type Station struct {
	mu      sync.Mutex      // verrou : empêche deux goroutines de modifier la station en même temps
	id      types.StationID // identifiant de la station
	node    types.NodeID    // nœud du graphe où se trouve la station
	pumps   []Pump          // les bornes
	queue   []ChargeRequest // la file d'attente (le premier arrivé est en position 0)
	price   types.Price     // prix en centimes d'euro par kWh
	revenue float64         // ce que la station a gagné (en centimes)
}

// NewStation crée une station avec nbPumps bornes de puissance powerKW
func NewStation(id types.StationID, node types.NodeID, nbPumps int, powerKW float64, price types.Price) *Station {
	pumps := make([]Pump, nbPumps)
	for i := range pumps {
		pumps[i].PowerKW = powerKW
	}
	return &Station{
		id:    id,
		node:  node,
		pumps: pumps,
		price: price,
	}
}

// Arrive : une voiture arrive à la station.
// Si une borne est libre (et pas en panne), on la branche, sinon elle va dans la file.
func (s *Station) Arrive(carID types.CarID, neededKWh float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := ChargeRequest{CarID: carID, NeededKWh: neededKWh}
	for i := range s.pumps {
		if !s.pumps[i].Broken && s.pumps[i].Car == nil {
			s.pumps[i].Car = &req
			return
		}
	}
	s.queue = append(s.queue, req)
}

// step : ce que fait la station pendant 1 pas de temps (= 1 minute).
// Renvoie la liste des voitures qui ont fini de se recharger.
func (s *Station) step(t types.SimTime) []types.CarID {
	s.mu.Lock()
	defer s.mu.Unlock()

	var finished []types.CarID

	// 1. Recharger les voitures branchées
	for i := range s.pumps {
		p := &s.pumps[i]
		if p.Broken || p.Car == nil {
			continue
		}
		energy := p.PowerKW / 60 // kWh donnés en 1 minute
		if energy > p.Car.NeededKWh {
			energy = p.Car.NeededKWh
		}
		p.Car.NeededKWh -= energy
		s.revenue += energy * float64(s.price)

		// La voiture est pleine : elle part, la borne se libère
		if p.Car.NeededKWh <= 0 {
			finished = append(finished, p.Car.CarID)
			p.Car = nil
		}
	}

	// 2. Faire avancer la file : brancher les voitures qui attendent sur les bornes libres
	for i := range s.pumps {
		if len(s.queue) == 0 {
			break
		}
		if !s.pumps[i].Broken && s.pumps[i].Car == nil {
			req := s.queue[0]
			s.queue = s.queue[1:]
			s.pumps[i].Car = &req
		}
	}

	return finished
}

// QueueLength : nombre de voitures dans la file
func (s *Station) QueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

// FreePumps : nombre de bornes libres et en état de marche
func (s *Station) FreePumps() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, p := range s.pumps {
		if !p.Broken && p.Car == nil {
			n++
		}
	}
	return n
}
