package gridmap

import (
	"github.com/dslaskibert/ia04-projet-sma/types"
)

// GridMap est une carte où la ville représente un grille régulière M×N
// toutes les arêtes ont la même longueur en mètres

type gridNode struct {
	id        types.NodeID
	x, y      int
	position  types.Position
	isStation bool
}

func (n *gridNode) ID() types.NodeID {
	return n.id
}

func (n *gridNode) Position() types.Position {
	return n.position
}

func (n *gridNode) IsStation() bool {
	return n.isStation
}

type gridEdge struct {
	from, to types.NodeID
	length   float64
}

func (e *gridEdge) From() types.NodeID {
	return e.from
}

func (e *gridEdge) To() types.NodeID {
	return e.to
}

func (e *gridEdge) Length() float64 {
	return e.length
}

type GridMap struct {
	width, height int
	edgeLength    float64
	nodes         []gridNode
	stations      []types.NodeID
}

func (m *GridMap) NodeCount() int {
	return len(m.nodes)
}
