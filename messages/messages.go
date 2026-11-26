package messages

import "github.com/dslaskibert/ia04-projet-sma/types"

// interface commune à tous les messages reçus par une station
type StationMessage interface {
	SenderID() types.CarID
}

// voiture qui vient d'arriver à une station et demande à rejoindre la file d'attente
type ArriveMessage struct {
	CarID     types.CarID
	NeededKWh float64           // quantité d'énergie demandée
	ReplyChan chan StationReply // channel sur lequel la station répond (QueuePositionMessage, ChargeFinishedMessage, ..)
	// la voiture doit créer ce channel bufferisé (buffer >= 2),
	// sinon la station bloquerait au moment d'écrire la réponse et figerait toute la simulation
}

func (m ArriveMessage) SenderID() types.CarID {
	return m.CarID
}

// voiture qui renonce à attendre (changement station ou autre selon stratégie)
type LeaveQueueMessage struct {
	CarID types.CarID
}

func (m LeaveQueueMessage) SenderID() types.CarID {
	return m.CarID
}

// voiture qui demande à interrompre une recharge en cours
type StopChargeMessage struct {
	CarID types.CarID
}

func (m StopChargeMessage) SenderID() types.CarID {
	return m.CarID
}

// station informe une voiture de sa position dans la file + estimation d'attente
// Convention : Position à 0 signifie que la voiture est branchée et en cours de recharge (elle n'attend plus)
// EstimatedWaitTicks vaut alors 0
type QueuePositionMessage struct {
	CarID              types.CarID
	Position           int
	EstimatedWaitTicks int
}

func (m QueuePositionMessage) SenderID() types.CarID { return m.CarID }

// station signale à une voiture que sa recharge est terminée + bilan
type ChargeFinishedMessage struct {
	CarID       types.CarID
	EnergyGiven float64
	Cost        types.Cost
}

// interface commune aux réponses envoyées par une station
type StationReply interface {
	ReplyFor() types.CarID
}

func (m QueuePositionMessage) ReplyFor() types.CarID {
	return m.CarID
}

func (m ChargeFinishedMessage) ReplyFor() types.CarID {
	return m.CarID
}
