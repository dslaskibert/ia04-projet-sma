package environment

import (
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/types"
)

// La carte est représentée par un graphe orienté avec des noeuds (nodes) et des arêtes (edges)

type Node interface {
	ID() types.NodeID
	Position() types.Position
	IsStation() bool
}

type Edge interface {
	From() types.NodeID
	To() types.NodeID
	Length() float64 // mètres
}

// Map est le graphe
type Map interface {
	NodeCount() int
	NodeByID(id types.NodeID) (Node, error)
	Edges(from types.NodeID) []Edge
	Stations() []types.NodeID // liste des noeuds stations
	// renvoie la séquence de noeuds du plus court chemin entre from et to avec la longueur totale en mètres
	ShortestPath(from types.NodeID, to types.NodeID) ([]types.NodeID, float64, error)
	RandomOriginDestination(rng *rand.Rand) (origin types.NodeID, destination types.NodeID) // trajet aléatoire entre deux noeuds du graphe
	AllNodes() []Node
	AllEdges() []Edge
}
