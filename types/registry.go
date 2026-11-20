package types

// annuaire des agents de la simulation
// permet aux agents de communiquer entre eux
type Registry interface {
	// la station associée à l'ID
	StationByID(id StationID) StationHandle

	// toutes les stations
	AllStations() []StationHandle

	// la voiture associée à l'ID
	CarByID(id CarID) CarHandle
}

// vue minimale qu'un agent a sur une station via le registre
type StationHandle interface {
	ID() StationID
	Position() NodeID
	Inbox() chan<- any // chan<- messages.StationMessage pour la station
}

// vue minimale qu'un agent a sur une voiture via le registre
type CarHandle interface {
	ID() CarID
	Position() NodeID
}
