package car


import (
	"github.com/dslaskibert/ia04-projet-sma/types"
)


type CarStrategy string

const (
	StrategyNearest	  		CarStrategy = "nearest"
	StrategyCheapest  		CarStrategy = "cheapest"
	StrategyShortestQueue 	CarStrategy = "shortest_queue"
	StrategyCombined 		CarStrategy = "combined"
)

func (c *Car) SetStrategy (strat CarStrategy) {
	c.strategy = strat
}

func (c *Car) chooseStation(stations []StationInfoView) (types.StationID, types.NodeID, []types.NodeID) {
	switch c.strategy {
	case StrategyCheapest:
		return c.chooseCheapestStation(stations)
	case StrategyShortestQueue:
		return c.chooseQueueStation(stations)
	case StrategyCombined:
		return c.chooseCombinedStation(stations)
	case StrategyNearest:
		fallthrough
	default:
		return c.chooseNearestStation(stations)
	}
}

func (c *Car) chooseNearestStation(stations []StationInfoView) (types.StationID, types.NodeID, []types.NodeID) {
	var bestStation types.StationID
	var bestStationNode types.NodeID
	var bestPath []types.NodeID
	var minDist float64 = -1

	for _, info := range stations {
		pathSt, dist, err := c.Map.ShortestPath(c.currentNode, info.NodeID)
		if err == nil && (minDist < 0 || dist < minDist) {
			minDist = dist
			bestStation = info.StationID
			bestStationNode = info.NodeID
			bestPath = pathSt
		}
	}
	return bestStation, bestStationNode, bestPath
}

func (c *Car) chooseCheapestStation(stations []StationInfoView) (types.StationID, types.NodeID, []types.NodeID) {
	var bestStation types.StationID
	var bestStationNode types.NodeID
	var bestPath []types.NodeID
	var minPrice types.Price = -1
	var bestDist float64 = -1
	carRange := c.Range()

	for _, info := range stations {
		pathSt, dist, err := c.Map.ShortestPath(c.currentNode, info.NodeID)
		if err != nil || dist > carRange {
			continue
		}

		if minPrice < 0 || info.Price < minPrice || (info.Price == minPrice && dist < bestDist) {
			minPrice = info.Price
			bestDist = dist
			bestStation = info.StationID
			bestStationNode = info.NodeID
			bestPath = pathSt
		}
	}

	if bestStation == 0 {
		return c.chooseNearestStation(stations)
	}

	return bestStation, bestStationNode, bestPath
}

func (c *Car) chooseQueueStation(stations []StationInfoView) (types.StationID, types.NodeID, []types.NodeID) {
	var bestStation types.StationID
	var bestStationNode types.NodeID
	var bestPath []types.NodeID
	var minQueue int = -1
	var bestDist float64 = -1
	carRange := c.Range()

	for _, info := range stations {
		pathSt, dist, err := c.Map.ShortestPath(c.currentNode, info.NodeID)
		if err != nil || dist > carRange {
			continue
		}

		if minQueue < 0 || info.QueueLength < minQueue || (info.QueueLength == minQueue && dist < bestDist) {
			minQueue = info.QueueLength
			bestDist = dist
			bestStation = info.StationID
			bestStationNode = info.NodeID
			bestPath = pathSt
		}
	}

	if bestStation == 0 {
		return c.chooseNearestStation(stations)
	}

	return bestStation, bestStationNode, bestPath
}

func (c *Car) chooseCombinedStation(stations []StationInfoView) (types.StationID, types.NodeID, []types.NodeID) {
	var bestStation types.StationID
	var bestStationNode types.NodeID
	var bestPath []types.NodeID
	var minScore float64 = -1
	carRange := c.Range()

	const (
		wDist  = 1.0  // cost per square (100 meters)
		wPrice = 50.0 // sensibility of the price (€/kWh)
		wQueue = 10.0 // cost for waiting each car on the queue
	)

	for _, info := range stations {
		pathSt, dist, err := c.Map.ShortestPath(c.currentNode, info.NodeID)
		if err != nil || dist > carRange {
			continue
		}

		score := (dist / 100.0) * wDist + float64(info.Price) * wPrice + float64(info.QueueLength) * wQueue

		if minScore < 0 || score < minScore {
			minScore = score
			bestStation = info.StationID
			bestStationNode = info.NodeID
			bestPath = pathSt
		}
	}

	if bestStation == 0 {
		return c.chooseNearestStation(stations)
	}

	return bestStation, bestStationNode, bestPath
}