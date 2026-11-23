package station

import (
	"fmt"
	"math"

	"github.com/dslaskibert/ia04-projet-sma/messages"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

// Vérifie que Station respecte l'interface StationHandle du registre de David
var _ types.StationHandle = (*Station)(nil)

// taille de la boîte aux lettres : assez grand pour qu'une voiture
// ne reste jamais bloquée quand elle envoie un message
const inboxSize = 1000

// chargeRequest : une voiture branchée ou dans la file, vue par la station
type chargeRequest struct {
	carID     types.CarID
	neededKWh float64                    // énergie qu'il reste à lui donner
	charged   float64                    // énergie déjà donnée
	reply     chan messages.StationReply // pour répondre à la voiture
}

// une borne de recharge de la station
type Pump struct {
	PowerKW float64        // puissance de la borne (en kW)
	Broken  bool           // true si la borne est en panne
	Car     *chargeRequest // voiture branchée (nil = borne libre)
}

// Station : l'agent station
type Station struct {
	id      types.StationID // identifiant de la station
	node    types.NodeID    // nœud du graphe où se trouve la station
	pumps   []Pump          // les bornes
	queue   []chargeRequest // la file d'attente (le premier arrivé est en position 0)
	price   types.Price     // prix en centimes d'euro par kWh
	revenue float64         // ce que la station a gagné (en centimes)
	inbox   chan any        // boîte aux lettres : les messages des voitures arrivent ici
}

// StationView : copie de l'état de la station, pour l'interface et les voitures
// (même principe que CarView dans car.go)
type StationView struct {
	ID          types.StationID
	Node        types.NodeID
	NbPumps     int
	FreePumps   int
	QueueLength int
	Price       types.Price
	Revenue     float64
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
		inbox: make(chan any, inboxSize),
	}
}

// ID, Position et Inbox : ce que le registre de David demande (StationHandle)

func (s *Station) ID() types.StationID {
	return s.id
}

func (s *Station) Position() types.NodeID {
	return s.node
}

// Inbox : le channel où les voitures envoient leurs messages (messages.ArriveMessage, ...).
// On renvoie un chan<- pour que les autres puissent seulement envoyer, pas lire.
func (s *Station) Inbox() chan<- any {
	return s.inbox
}

// receiveMessages : on vide la boîte aux lettres au début du tour
// et on traite chaque message selon son type
func (s *Station) receiveMessages(t types.SimTime) {
	for {
		select {
		case msg := <-s.inbox:
			switch m := msg.(type) {
			case messages.ArriveMessage:
				s.arrive(m, t)
			case messages.LeaveQueueMessage:
				s.leaveQueue(m.CarID, t)
			default:
				fmt.Printf("[%s] tour %d : message inconnu %T, ignoré\n", s.Name(), t, msg)
			}
		default:
			return // la boîte est vide
		}
	}
}

// arrive : une voiture arrive à la station.
// Si une borne est libre (et pas en panne), on la branche, sinon elle va dans la file.
// Dans les deux cas on lui répond avec sa position :
// 0 = branchée tout de suite, 1 = première de la file, etc.
func (s *Station) arrive(m messages.ArriveMessage, t types.SimTime) {
	req := chargeRequest{carID: m.CarID, neededKWh: m.NeededKWh, reply: m.ReplyChan}

	for i := range s.pumps {
		if !s.pumps[i].Broken && s.pumps[i].Car == nil {
			s.pumps[i].Car = &req
			fmt.Printf("[%s] tour %d : voiture %d branchée\n", s.Name(), t, m.CarID)
			s.notify(req.reply, messages.QueuePositionMessage{CarID: m.CarID, Position: 0, EstimatedWaitTicks: 0})
			return
		}
	}

	s.queue = append(s.queue, req)
	pos := len(s.queue)
	wait := s.estimateWait(pos)
	fmt.Printf("[%s] tour %d : voiture %d dans la file (position %d, attente estimée %d tours)\n", s.Name(), t, m.CarID, pos, wait)
	s.notify(req.reply, messages.QueuePositionMessage{CarID: m.CarID, Position: pos, EstimatedWaitTicks: wait})
}

// leaveQueue : une voiture en a marre d'attendre et quitte la file
func (s *Station) leaveQueue(carID types.CarID, t types.SimTime) {
	for i, req := range s.queue {
		if req.carID == carID {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			fmt.Printf("[%s] tour %d : voiture %d quitte la file\n", s.Name(), t, carID)
			return
		}
	}
	// pas dans la file (déjà branchée ou inconnue) : on ne fait rien
	fmt.Printf("[%s] tour %d : voiture %d veut quitter la file mais n'y est pas\n", s.Name(), t, carID)
}

// estimateWait : estime combien de tours va attendre la voiture en position pos de la file.
// On calcule dans combien de minutes chaque borne sera libre, puis on "place" les voitures
// de la file une par une sur la borne qui se libère le plus tôt.
// Renvoie -1 si toutes les bornes sont en panne.
func (s *Station) estimateWait(pos int) int {
	var free []float64  // pour chaque borne qui marche : dans combien de minutes elle est libre
	var power []float64 // et sa puissance
	for _, p := range s.pumps {
		if p.Broken {
			continue
		}
		minutes := 0.0
		if p.Car != nil {
			minutes = p.Car.neededKWh / p.PowerKW * 60
		}
		free = append(free, minutes)
		power = append(power, p.PowerKW)
	}
	if len(free) == 0 {
		return -1
	}

	for i := 0; i < pos && i < len(s.queue); i++ {
		// la borne qui se libère le plus tôt
		k := 0
		for j := range free {
			if free[j] < free[k] {
				k = j
			}
		}
		if i == pos-1 {
			// arrondi au tour supérieur (le -1e-9 c'est encore pour les arrondis des float)
			return int(math.Ceil(free[k] - 1e-9))
		}
		free[k] += s.queue[i].neededKWh / power[k] * 60
	}
	return 0
}

// step : ce que fait la station pendant 1 pas de temps (= 1 minute).
// Renvoie les recharges qui se sont terminées pendant ce tour.
func (s *Station) step(_ types.SimTime) []messages.ChargeFinishedMessage {
	var finished []messages.ChargeFinishedMessage

	// 1. Recharger les voitures branchées
	for i := range s.pumps {
		p := &s.pumps[i]
		if p.Broken || p.Car == nil {
			continue
		}
		energy := p.PowerKW / 60 // kWh donnés en 1 minute
		if energy > p.Car.neededKWh {
			energy = p.Car.neededKWh
		}
		p.Car.neededKWh -= energy
		p.Car.charged += energy
		s.revenue += energy * float64(s.price)

		// La voiture est pleine : elle part, la borne se libère.
		// On compare à 1e-9 et pas à 0 à cause des arrondis des float
		// (sinon 30 kWh prenait 37 tours au lieu de 36)
		if p.Car.neededKWh <= 1e-9 {
			msg := messages.ChargeFinishedMessage{
				CarID:       p.Car.carID,
				EnergyGiven: p.Car.charged,
				Cost:        types.Price(p.Car.charged * float64(s.price)),
			}
			s.notify(p.Car.reply, msg)
			finished = append(finished, msg)
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

// notify : envoie une réponse à la voiture.
// Si le channel de la voiture est plein (ou sans buffer), on ne bloque pas la station
// (sinon toute la simu se bloque), on affiche juste un message.
func (s *Station) notify(ch chan messages.StationReply, msg messages.StationReply) {
	if ch == nil {
		return
	}
	select {
	case ch <- msg:
	default:
		fmt.Printf("[%s] attention : impossible de prévenir la voiture %d (channel plein ?)\n", s.Name(), msg.ReplyFor())
	}
}

// freePumps : nombre de bornes libres et en état de marche
func (s *Station) freePumps() int {
	n := 0
	for _, p := range s.pumps {
		if !p.Broken && p.Car == nil {
			n++
		}
	}
	return n
}

// Snapshot : renvoie une copie de l'état de la station.
// À appeler entre deux tours (quand la station ne travaille pas), comme pour les voitures.
func (s *Station) Snapshot() StationView {
	return StationView{
		ID:          s.id,
		Node:        s.node,
		NbPumps:     len(s.pumps),
		FreePumps:   s.freePumps(),
		QueueLength: len(s.queue),
		Price:       s.price,
		Revenue:     s.revenue,
	}
}
