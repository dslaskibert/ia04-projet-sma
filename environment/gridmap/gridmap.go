package gridmap

import (
	"container/heap"
	"fmt"
	"math"
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

// Élément de la priority queue
type pqItem struct {
	id       types.NodeID
	distance float64
}

// File de priorité
type priorityQueue []*pqItem

func (pq priorityQueue) Len() int {
	return len(pq)
}

// Comparaison plus petite distance
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].distance < pq[j].distance
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*pqItem)
	*pq = append(*pq, item)
}

// Retrait du premier élément
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)

	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]

	return item
}

func (m *GridMap) ShortestPath(from types.NodeID, to types.NodeID) ([]types.NodeID, float64, error) {
	n := len(m.nodes)

	// Vérifier la validité
	if int(from) < 0 || int(from) >= n {
		return nil, 0, fmt.Errorf("invalid origin node: %v", from)
	}

	if int(to) < 0 || int(to) >= n {
		return nil, 0, fmt.Errorf("invalid destination node: %v", to)
	}

	// si départ = arrivée
	if from == to {
		return []types.NodeID{from}, 0, nil
	}

	// Initialisation des distances et prédécesseurs
	distances := make([]float64, n)
	predecessors := make([]types.NodeID, n)

	for i := range n {
		distances[i] = math.Inf(1)
		predecessors[i] = types.NodeID(-1)
	}

	distances[int(from)] = 0

	// priority queue
	pq := make(priorityQueue, 0)

	heap.Push(&pq, &pqItem{
		id:       from,
		distance: 0,
	})

	// Boucle dijkstra
	for pq.Len() > 0 {

		// noeud le plus proche
		current := heap.Pop(&pq).(*pqItem)

		u := current.id
		uIndex := int(u)

		// ignorer les anciennes entrées de la queue
		if current.distance > distances[uIndex] {
			continue
		}

		// plus court chemin est trouvé
		if u == to {
			break
		}

		// Explorer les voisins
		for _, edge := range m.Edges(u) {
			v := edge.To()
			vIndex := int(v)

			// Vérifier que le poids est valide
			weight := edge.Length()
			if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
				return nil, 0, fmt.Errorf("invalid edge weight from %v to %v", u, v)
			}

			newDistance := distances[uIndex] + weight

			// Si on trouve un chemin plus court
			if newDistance < distances[vIndex] {

				distances[vIndex] = newDistance
				predecessors[vIndex] = u

				// Ajouter le voisin à la priority queue
				heap.Push(&pq, &pqItem{
					id:       v,
					distance: newDistance,
				})
			}
		}
	}

	// Vérifier qu'un chemin existe
	if math.IsInf(distances[int(to)], 1) {
		return nil, 0, fmt.Errorf("no path found between %v and %v", from, to)
	}

	// Reconstruction du chemin
	path := make([]types.NodeID, 0)
	current := to

	for current != from {
		path = append(path, current)

		current = predecessors[int(current)]

		if current < 0 {
			return nil, 0, fmt.Errorf("failed to reconstruct path")
		}
	}

	// Ajouter le noeud de départ
	path = append(path, from)

	// Inverser le chemin (on est parti de l'arrivée)
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, distances[int(to)], nil
}

func (m *GridMap) RandomOriginDestination(rng *rand.Rand) (origin types.NodeID, destination types.NodeID) { // trajet aléatoire entre deux noeuds du graphe
	if len(m.nodes) < 2 {
		return 0, 0
	}

	start := rng.Intn(len(m.nodes))

	finish := rng.Intn(len(m.nodes))
	for finish == start {
		finish = rng.Intn(len(m.nodes))
	}

	return types.NodeID(start), types.NodeID(finish)
}

func (m *GridMap) AllNodes() []environment.Node {
	result := make([]environment.Node, len(m.nodes))
	for i := range m.nodes {
		result[i] = &m.nodes[i]
	}
	return result
}

func (m *GridMap) AllEdges() []environment.Edge {
	edges := make([]environment.Edge, 0)

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			fromID := types.NodeID(y*m.width + x)
			for _, e := range m.Edges(fromID) {
				if e.From() < e.To() { // éviter les doublons
					edges = append(edges, e)
				}
			}
		}
	}
	return edges
}

func (m *GridMap) Print() {
	for y := 0; y < m.height; y++ {

		// afficher les noeuds
		for x := 0; x < m.width; x++ {
			node := m.nodes[y*m.width+x]

			if node.isStation {
				fmt.Print("  *  ")
			} else {
				fmt.Print("  o  ")
			}

			if x < m.width-1 {
				fmt.Print("-----")
			}
		}
		fmt.Println()

		// afficher les rues verticales
		if y < m.height-1 {
			for x := 0; x < m.width; x++ {
				fmt.Print("  |  ")

				if x < m.width-1 {
					fmt.Print("     ")
				}
			}
			fmt.Println()
		}
	}
}
