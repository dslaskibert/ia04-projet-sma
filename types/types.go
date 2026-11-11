package types

// types agents
type AgentType string

const (
	AgentTypeCar     AgentType = "car"
	AgentTypeStation AgentType = "station"
)

// types ID
type CarID int
type StationID int

// type temps
type SimTime int // temps écoulé depuis le début en minutes

// récupérer le nombre de jours de la simulation
func (minutes SimTime) Day() int {
	return int(minutes / (60 * 24))
}

// récupérer la minute du jour en cours
func (minutes SimTime) MinuteOfDay() int {
	return int(minutes % (60 * 24))
}

// type prix
type Price float64 // prix en centimes d'euros /kWh

// types du graphe représentant la carte
type NodeID int

type Position struct {
	X float64
	Y float64
}
