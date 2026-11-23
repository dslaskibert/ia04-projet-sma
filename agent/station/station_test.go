package station

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/dslaskibert/ia04-projet-sma/messages"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

// tour : fait faire un tour à la station "à la main", comme le ferait Start
func tour(s *Station, t types.SimTime) []messages.ChargeFinishedMessage {
	s.receiveMessages(t)
	return s.step(t)
}

// reponses : récupère tout ce que la station a envoyé à une voiture (sans bloquer)
func reponses(ch chan messages.StationReply) []messages.StationReply {
	var out []messages.StationReply
	for {
		select {
		case r := <-ch:
			out = append(out, r)
		default:
			return out
		}
	}
}

// arrivee : le message qu'une voiture envoie en arrivant
func arrivee(id types.CarID, kwh float64, reply chan messages.StationReply) messages.ArriveMessage {
	return messages.ArriveMessage{CarID: id, NeededKWh: kwh, ReplyChan: reply}
}

// presque : pour comparer des float sans souci d'arrondi
func presque(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// Une voiture qui demande 30 kWh sur une borne de 50 kW doit mettre 36 minutes
func TestRecharge(t *testing.T) {
	s := NewStation(1, 0, 1, 50, 30)
	reply := make(chan messages.StationReply, 1)
	s.Inbox() <- arrivee(1, 30, reply)

	fin := 0
	for tick := 1; tick <= 100 && fin == 0; tick++ {
		tour(s, types.SimTime(tick))
		for _, r := range reponses(reply) {
			switch m := r.(type) {
			case messages.QueuePositionMessage:
				// au tour 1 la station doit dire "branchée" (position 0)
				if tick != 1 || m.Position != 0 {
					t.Errorf("tour %d : position %d, attendu position 0 au tour 1", tick, m.Position)
				}
			case messages.ChargeFinishedMessage:
				fin = tick
				if !presque(m.EnergyGiven, 30) || !presque(float64(m.Cost), 900) {
					t.Errorf("mauvais bilan : %+v", m)
				}
			}
		}
	}
	if fin != 36 {
		t.Errorf("recharge finie au tour %d, attendu 36", fin)
	}
	if !presque(s.Snapshot().Revenue, 900) {
		t.Errorf("revenu = %.2f, attendu 900", s.Snapshot().Revenue)
	}
	if s.Snapshot().FreePumps != 1 {
		t.Errorf("la borne devrait être libre après la recharge")
	}
}

// Avec une seule borne, la 2e voiture attend dans la file puis passe après la 1re
func TestFileAttente(t *testing.T) {
	s := NewStation(1, 0, 1, 50, 30)
	r2 := make(chan messages.StationReply, 1)
	s.Inbox() <- arrivee(1, 30, nil)
	s.Inbox() <- arrivee(2, 30, r2)

	tour(s, 1)
	if s.Snapshot().QueueLength != 1 {
		t.Fatalf("file = %d, attendu 1", s.Snapshot().QueueLength)
	}
	// la voiture 2 est 1re de la file et doit attendre que la voiture 1 finisse (36 tours)
	rep := reponses(r2)
	if len(rep) != 1 {
		t.Fatalf("la voiture 2 devrait avoir reçu sa position")
	}
	if m, ok := rep[0].(messages.QueuePositionMessage); !ok || m.Position != 1 || m.EstimatedWaitTicks != 36 {
		t.Errorf("réponse = %+v, attendu position 1 et 36 tours d'attente", rep[0])
	}

	finVoiture := map[types.CarID]int{}
	for tick := 2; tick <= 100; tick++ {
		for _, d := range tour(s, types.SimTime(tick)) {
			finVoiture[d.CarID] = tick
		}
	}
	if finVoiture[1] != 36 || finVoiture[2] != 72 {
		t.Errorf("fins = %v, attendu voiture 1 au tour 36 et voiture 2 au tour 72", finVoiture)
	}
	if s.Snapshot().QueueLength != 0 {
		t.Errorf("la file devrait être vide à la fin")
	}
}

// Une voiture qui quitte la file laisse sa place à la suivante
func TestQuitterFile(t *testing.T) {
	s := NewStation(1, 0, 1, 50, 30)
	r3 := make(chan messages.StationReply, 1)
	s.Inbox() <- arrivee(1, 30, nil)
	s.Inbox() <- arrivee(2, 30, nil)
	s.Inbox() <- arrivee(3, 30, r3)
	tour(s, 1)

	// la voiture 3 est 2e de la file : elle attend la voiture 1 puis la voiture 2
	if m, ok := reponses(r3)[0].(messages.QueuePositionMessage); !ok || m.Position != 2 || m.EstimatedWaitTicks != 72 {
		t.Errorf("voiture 3 : attendu position 2 et 72 tours d'attente")
	}

	s.Inbox() <- messages.LeaveQueueMessage{CarID: 2}
	tour(s, 2)
	if s.Snapshot().QueueLength != 1 {
		t.Errorf("file = %d, attendu 1 après le départ de la voiture 2", s.Snapshot().QueueLength)
	}

	finVoiture := map[types.CarID]int{}
	for tick := 3; tick <= 100; tick++ {
		for _, d := range tour(s, types.SimTime(tick)) {
			finVoiture[d.CarID] = tick
		}
	}
	if finVoiture[3] != 72 || finVoiture[2] != 0 {
		t.Errorf("fins = %v, attendu voiture 3 au tour 72 et rien pour la voiture 2", finVoiture)
	}
}

// Une borne en panne ne doit pas être utilisée
func TestBorneEnPanne(t *testing.T) {
	s := NewStation(1, 0, 2, 50, 30)
	s.pumps[0].Broken = true

	s.Inbox() <- arrivee(1, 10, nil)
	tour(s, 1)
	if s.pumps[0].Car != nil {
		t.Errorf("une voiture a été branchée sur la borne en panne")
	}
	if s.pumps[1].Car == nil || s.pumps[1].Car.carID != 1 {
		t.Errorf("la voiture 1 devrait être sur la borne 2")
	}

	// la 2e voiture doit attendre, même s'il y a "2 bornes"
	s.Inbox() <- arrivee(2, 10, nil)
	tour(s, 2)
	v := s.Snapshot()
	if v.FreePumps != 0 || v.QueueLength != 1 {
		t.Errorf("bornes libres = %d, file = %d, attendu 0 et 1", v.FreePumps, v.QueueLength)
	}
}

// Si toutes les bornes sont en panne, l'attente estimée vaut -1
func TestToutEnPanne(t *testing.T) {
	s := NewStation(1, 0, 1, 50, 30)
	s.pumps[0].Broken = true
	reply := make(chan messages.StationReply, 1)
	s.Inbox() <- arrivee(1, 10, reply)
	tour(s, 1)

	if m, ok := reponses(reply)[0].(messages.QueuePositionMessage); !ok || m.EstimatedWaitTicks != -1 {
		t.Errorf("attendu une attente de -1 quand tout est en panne")
	}
}

// Si la voiture oublie le buffer sur son channel, la station ne doit pas se bloquer
func TestChannelSansBuffer(t *testing.T) {
	s := NewStation(1, 0, 1, 60, 30)
	reply := make(chan messages.StationReply) // pas de buffer
	s.Inbox() <- arrivee(1, 1, reply)

	fini := tour(s, 1) // 60 kW pendant 1 minute = 1 kWh, donc fini en 1 tour
	if len(fini) != 1 {
		t.Errorf("la recharge devrait être finie au tour 1")
	}
}

// Un message que la station ne connaît pas est ignoré
func TestMessageInconnu(t *testing.T) {
	s := NewStation(1, 0, 1, 50, 30)
	s.Inbox() <- "bonjour"
	tour(s, 1)
	if s.Snapshot().QueueLength != 0 || s.Snapshot().FreePumps != 1 {
		t.Errorf("le message inconnu n'aurait rien dû changer")
	}
}

// La station dans sa goroutine, avec les channels du moteur et le registre de David
func TestStart(t *testing.T) {
	s := NewStation(1, 0, 4, 50, 30)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tickIn := make(chan types.SimTime)
	doneOut := make(chan struct{})
	go s.Start(ctx, tickIn, doneOut)

	// la voiture passe par StationHandle, comme si elle avait trouvé la station dans le registre
	var h types.StationHandle = s
	reply := make(chan messages.StationReply, 1)
	h.Inbox() <- arrivee(7, 10, reply)

	for tick := 1; tick <= 20; tick++ {
		tickIn <- types.SimTime(tick)
		select {
		case <-doneOut:
		case <-time.After(time.Second):
			t.Fatalf("la station n'a pas répondu au tour %d", tick)
		}

		for _, r := range reponses(reply) {
			if m, ok := r.(messages.ChargeFinishedMessage); ok {
				// 10 kWh à 50 kW = 12 minutes
				if tick != 12 || m.CarID != 7 {
					t.Errorf("voiture %d rechargée au tour %d, attendu voiture 7 au tour 12", m.CarID, tick)
				}
				return
			}
		}
	}
	t.Errorf("la voiture n'a jamais été prévenue")
}
