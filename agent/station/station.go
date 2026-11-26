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
	cost      types.Cost                 // ce qu'elle doit déjà payer (en euros), compté à chaque tour
	pos       int                        // dernière position qu'on lui a annoncée (0 = branchée)
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
	queue   []chargeRequest // la file d'attente (le premier arrivé est à l'indice 0, c'est la position 1 pour la voiture)
	price   types.Price     // prix en centimes d'euro par kWh
	revenue types.Cost      // ce que la station a gagné (en euros)
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
	Revenue     types.Cost // en euros
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
			case messages.StopChargeMessage:
				s.stopCharge(m.CarID, t)
			default:
				fmt.Printf("[%s] tour %d : message inconnu %T, ignoré\n", s.Name(), t, msg)
			}
		default:
			return // la boîte est vide
		}
	}
}

// arrive : une voiture arrive à la station.
// Si personne n'attend et qu'une borne est libre (et pas en panne), on la branche,
// sinon elle va dans la file (elle ne passe pas devant celles qui attendent déjà).
// Dans les deux cas on lui répond avec sa position :
// 0 = branchée tout de suite, 1 = première de la file, etc.
func (s *Station) arrive(m messages.ArriveMessage, t types.SimTime) {
	req := chargeRequest{carID: m.CarID, neededKWh: m.NeededKWh, reply: m.ReplyChan}

	if len(s.queue) == 0 {
		for i := range s.pumps {
			if !s.pumps[i].Broken && s.pumps[i].Car == nil {
				s.pumps[i].Car = &req
				fmt.Printf("[%s] tour %d : voiture %d branchée\n", s.Name(), t, m.CarID)
				s.notify(req.reply, messages.QueuePositionMessage{CarID: m.CarID, Position: 0, EstimatedWaitTicks: 0})
				return
			}
		}
	}

	req.pos = len(s.queue) + 1
	s.queue = append(s.queue, req)
	wait := s.estimateWait(req.pos)
	fmt.Printf("[%s] tour %d : voiture %d dans la file (position %d, attente estimée %d tours)\n", s.Name(), t, m.CarID, req.pos, wait)
	s.notify(req.reply, messages.QueuePositionMessage{CarID: m.CarID, Position: req.pos, EstimatedWaitTicks: wait})
}

// leaveQueue : une voiture en a marre d'attendre et quitte la file.
// Les voitures derrière elle avancent : elles sont prévenues à la fin du tour (sendPositions).
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

// stopCharge : une voiture veut partir avant la fin de sa recharge.
// Si elle est branchée, on arrête : elle reçoit le bilan de ce qu'elle a eu
// (ChargeFinishedMessage avec l'énergie déjà donnée et le coût) et la borne se libère.
// Si elle est encore dans la file, c'est comme quitter la file.
func (s *Station) stopCharge(carID types.CarID, t types.SimTime) {
	for i := range s.pumps {
		if s.pumps[i].Car != nil && s.pumps[i].Car.carID == carID {
			msg := s.finishCharge(i)
			fmt.Printf("[%s] tour %d : voiture %d arrête sa recharge (%.1f kWh, %.2f €)\n",
				s.Name(), t, carID, msg.EnergyGiven, float64(msg.Cost))
			return
		}
	}
	s.leaveQueue(carID, t)
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
		// on compte le coût à chaque tour, au prix du moment
		// (comme ça ce sera juste quand le prix changera pendant une recharge)
		p.Car.cost += s.cost(energy)
		s.revenue += s.cost(energy)

		// La voiture est pleine : elle part, la borne se libère.
		// On compare à 1e-9 et pas à 0 à cause des arrondis des float
		// (sinon 30 kWh prenait 37 tours au lieu de 36)
		if p.Car.neededKWh <= 1e-9 {
			finished = append(finished, s.finishCharge(i))
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

	// 3. Prévenir les voitures dont la position a changé pendant ce tour
	s.sendPositions()

	return finished
}

// finishCharge : la voiture de la borne i a fini (ou part avant la fin).
// On lui envoie le bilan et on libère la borne.
func (s *Station) finishCharge(i int) messages.ChargeFinishedMessage {
	car := s.pumps[i].Car
	msg := messages.ChargeFinishedMessage{
		CarID:       car.carID,
		EnergyGiven: car.charged,
		Cost:        car.cost,
	}
	s.notify(car.reply, msg)
	s.pumps[i].Car = nil
	return msg
}

// sendPositions : à la fin du tour, on prévient chaque voiture dont la position a changé.
// Une voiture de la file qui vient d'être branchée reçoit la position 0. Une voiture qui a
// avancé dans la file (quelqu'un devant elle a été branché ou est parti) reçoit sa nouvelle
// position et la nouvelle attente estimée.
// On le fait une seule fois, en fin de tour, et pas à chaque changement : une voiture reçoit
// ainsi au plus 2 messages par tour, ce qui tient dans le buffer de 2 demandé dans messages.go.
// Sinon, si plusieurs voitures partent au même tour, le channel se remplit et c'est la
// position la plus récente (la bonne) qui serait perdue.
func (s *Station) sendPositions() {
	for i := range s.pumps {
		car := s.pumps[i].Car
		if car != nil && car.pos != 0 {
			car.pos = 0
			s.notify(car.reply, messages.QueuePositionMessage{CarID: car.carID, Position: 0, EstimatedWaitTicks: 0})
		}
	}
	for j := range s.queue {
		pos := j + 1
		if s.queue[j].pos != pos {
			s.queue[j].pos = pos
			s.notify(s.queue[j].reply, messages.QueuePositionMessage{
				CarID:              s.queue[j].carID,
				Position:           pos,
				EstimatedWaitTicks: s.estimateWait(pos),
			})
		}
	}
}

// cost : prix de kWh kilowattheures, en euros (le prix de la station est en centimes par kWh)
func (s *Station) cost(kWh float64) types.Cost {
	return types.Cost(kWh * float64(s.price) / 100)
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
