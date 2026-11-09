package gridmap

import (
	"fmt"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/environment"
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

func NewGridMap(width, height int, edgeLength float64, stationPositions [][2]int) *GridMap {
	stationSet := make(map[[2]int]bool, len(stationPositions))
	for _, pos := range stationPositions {
		stationSet[pos] = true
	}

	totalNodes := width * height
	nodes := make([]gridNode, 0, totalNodes)
	stations := make([]types.NodeID, 0, len(stationPositions))

	// Parcours de la grille en y puis en x et ID = y*width + x
	for y := range height {
		for x := range width {
			id := types.NodeID(y*width + x)
			isStation := stationSet[[2]int{x, y}]

			nodes = append(nodes, gridNode{
				id:        id,
				x:         x,
				y:         y,
				position:  types.Position{X: float64(x) * edgeLength, Y: float64(y) * edgeLength},
				isStation: isStation,
			})

			if isStation {
				stations = append(stations, id)
			}
		}
	}

	return &GridMap{
		width:      width,
		height:     height,
		edgeLength: edgeLength,
		nodes:      nodes,
		stations:   stations,
	}
}

func (m *GridMap) NodeCount() int {
	return len(m.nodes)
}

func (m *GridMap) NodeByID(id types.NodeID) (environment.Node, error) {
	for i := range m.nodes {
		if m.nodes[i].id == id {
			return &m.nodes[i], nil
		}
	}
	return nil, fmt.Errorf("node with ID %v not found", id)
}

func (m *GridMap) Edges(from types.NodeID) []environment.Edge {
	node := m.nodes[from]
	x, y := node.x, node.y

	directions := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	edges := make([]environment.Edge, 0, 4)

	for _, d := range directions {
		x2, y2 := x+d[0], y+d[1]
		if x2 < 0 || x2 >= m.width || y2 < 0 || y2 >= m.height {
			continue
		}
		toID := types.NodeID(y2*m.width + x2)
		edges = append(edges, &gridEdge{
			from:   from,
			to:     toID,
			length: m.edgeLength,
		})
	}

	return edges
}

func (m *GridMap) Stations() []types.NodeID { // liste des noeuds stations
	return m.stations
}

// TODO
func (m *GridMap) ShortestPath(from types.NodeID, to types.NodeID) ([]types.NodeID, float64, error) {
	return nil, 0, nil
}

// TODO
func (m *GridMap) RandomOriginDestination(rng *rand.Rand) (origin types.NodeID, destination types.NodeID) { // trajet aléatoire entre deux noeuds du graphe
	return 0, 0
}

func (m *GridMap) AllNodes() []environment.Node {
	result := make([]environment.Node, len(m.nodes))
	for i := range m.nodes {
		result[i] = &m.nodes[i]
	}
	return result
}

// TODO
func (m *GridMap) AllEdges() []environment.Edge {
	return nil
}
