// Package main : convertisseur de données OSM brutes vers un format simplifié
// de graphe nœuds+arêtes utilisable par la simulation.
//
// Lecture : data/compiegne_raw.json (sortie Overpass en JSON OSM)
// Écriture : data/compiegne.json (notre format simplifié)
//
// Lancement : go run ./cmd/osm2map
// Avec chemins custom : go run ./cmd/osm2map input.json output.json
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
)

// --- Types d'entrée (format Overpass OSM JSON) ---

type OSMElement struct {
	Type  string            `json:"type"`
	ID    int64             `json:"id"`
	Lat   float64           `json:"lat,omitempty"`   // seulement pour les nodes
	Lon   float64           `json:"lon,omitempty"`   // seulement pour les nodes
	Nodes []int64           `json:"nodes,omitempty"` // seulement pour les ways (liste d'OSM IDs)
	Tags  map[string]string `json:"tags,omitempty"`
}

type OSMData struct {
	Elements []OSMElement `json:"elements"`
}

// --- Types de sortie (notre format simplifié) ---

type MapNode struct {
	ID    int     `json:"id"`     // ID séquentiel interne (0, 1, 2, ...)
	OSMID int64   `json:"osm_id"` // ID OSM d'origine (pour debug)
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

type MapEdge struct {
	From    int     `json:"from"`
	To      int     `json:"to"`
	LengthM float64 `json:"length_m"`
	Name    string  `json:"name,omitempty"`
}

type MapData struct {
	Nodes []MapNode `json:"nodes"`
	Edges []MapEdge `json:"edges"`
}

// haversine renvoie la distance en mètres entre deux points (lat, lon) en degrés.
// Formule standard pour la distance orthodromique sur une sphère terrestre.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

func main() {
	inputPath := "data/compiegne_raw.json"
	outputPath := "data/compiegne.json"
	if len(os.Args) > 1 {
		inputPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputPath = os.Args[2]
	}

	fmt.Printf("Lecture de %s\n", inputPath)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.Fatalf("Lecture %s: %v", inputPath, err)
	}

	var osm OSMData
	if err := json.Unmarshal(data, &osm); err != nil {
		log.Fatalf("Parse JSON: %v", err)
	}

	// Séparation : d'un côté les nœuds (avec leurs coordonnées), de l'autre les ways
	osmNodes := make(map[int64]OSMElement)
	var ways []OSMElement
	for _, el := range osm.Elements {
		switch el.Type {
		case "node":
			osmNodes[el.ID] = el
		case "way":
			if len(el.Nodes) >= 2 {
				ways = append(ways, el)
			}
		}
	}
	fmt.Printf("OSM parsé : %d nœuds, %d chemins valides\n", len(osmNodes), len(ways))

	// Identification des nœuds vraiment référencés par au moins un way qu'on garde.
	// Un nœud OSM orphelin (pas référencé) est inutile pour notre simu.
	usedOSMNodes := make(map[int64]bool)
	for _, way := range ways {
		for _, nid := range way.Nodes {
			usedOSMNodes[nid] = true
		}
	}

	// Renumérotation : OSM ID (int64, gros chiffres) -> ID interne séquentiel (0, 1, 2, ...)
	// Avantages : IDs contigus pour indexer des slices, plus compact, lisible.
	osmToLocal := make(map[int64]int)
	var nodes []MapNode
	for osmID := range usedOSMNodes {
		osmNode, ok := osmNodes[osmID]
		if !ok {
			// Ref vers un nœud qu'on n'a pas récupéré (coupure au bord de la bbox)
			continue
		}
		localID := len(nodes)
		osmToLocal[osmID] = localID
		nodes = append(nodes, MapNode{
			ID:    localID,
			OSMID: osmID,
			Lat:   osmNode.Lat,
			Lon:   osmNode.Lon,
		})
	}
	fmt.Printf("Nœuds retenus (avec coordonnées valides) : %d\n", len(nodes))

	// Construction des arêtes à partir des ways.
	// Chaque way est une séquence de nœuds : on crée une arête entre chaque paire consécutive.
	// Dédup : la clé [min, max] évite de stocker la même arête deux fois quand deux ways
	// se chevauchent (très fréquent aux intersections).
	edgeSet := make(map[[2]int]MapEdge)
	for _, way := range ways {
		name := way.Tags["name"]
		for i := 0; i < len(way.Nodes)-1; i++ {
			from, ok1 := osmToLocal[way.Nodes[i]]
			to, ok2 := osmToLocal[way.Nodes[i+1]]
			if !ok1 || !ok2 {
				continue
			}
			if from == to {
				continue // auto-boucle, rare mais possible sur certaines données OSM
			}
			// Clé normalisée pour que (A, B) et (B, A) soient la même arête
			a, b := from, to
			if a > b {
				a, b = b, a
			}
			key := [2]int{a, b}
			if _, exists := edgeSet[key]; exists {
				continue
			}
			edgeSet[key] = MapEdge{
				From:    a,
				To:      b,
				LengthM: haversine(nodes[a].Lat, nodes[a].Lon, nodes[b].Lat, nodes[b].Lon),
				Name:    name,
			}
		}
	}

	edges := make([]MapEdge, 0, len(edgeSet))
	for _, e := range edgeSet {
		edges = append(edges, e)
	}

	// Stats pour vérifier que ça a l'air correct
	var totalLenM float64
	var minLen, maxLen float64 = math.Inf(1), 0
	for _, e := range edges {
		totalLenM += e.LengthM
		if e.LengthM < minLen {
			minLen = e.LengthM
		}
		if e.LengthM > maxLen {
			maxLen = e.LengthM
		}
	}
	fmt.Printf("Arêtes créées (dédupliquées) : %d\n", len(edges))
	fmt.Printf("Longueur totale du réseau : %.2f km\n", totalLenM/1000)
	fmt.Printf("Longueur moyenne par arête : %.1f m\n", totalLenM/float64(len(edges)))
	fmt.Printf("Longueur min/max : %.1f m / %.1f m\n", minLen, maxLen)

	// Sérialisation + écriture
	out := MapData{Nodes: nodes, Edges: edges}
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		log.Fatalf("Serialize JSON: %v", err)
	}
	if err := os.WriteFile(outputPath, buf, 0644); err != nil {
		log.Fatalf("Écriture %s: %v", outputPath, err)
	}
	fmt.Printf("Écrit dans %s (%.1f Ko)\n", outputPath, float64(len(buf))/1024)
}
