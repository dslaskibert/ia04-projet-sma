package car

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/environment"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

var _ agent.Agent = (*Car)(nil)

type CarState string

const (
	StateInRoute            CarState = "IN_ROUTE"
	StateToStation          CarState = "TO_STATION"
	StateAtStation          CarState = "AT_STATION"
	StateInLine             CarState = "IN_LINE"
	StateCharging           CarState = "CHARGING"
	StateArrivedDestination CarState = "ARRIVED_DESTINATION"
	StateBrokenDown         CarState = "BROKEN_DOWN"
)

type CarPerception struct {
	CurrentTick  types.SimTime
	NodePosition types.NodeID
	BatteryLevel float64
	StationInfos []StationInfoView
}

type StationInfoView struct {
	StationID   types.StationID
	NodeID      types.NodeID
	QueueLength int
	Price       types.Price
	Available   bool
}

// what the car can do in the turn
type ActionType string

const (
	ActionMove        ActionType = "MOVE"
	ActionGoToStation ActionType = "GO_TO_STATION"
	ActionEnterQueue  ActionType = "ENTER_QUEUE"
	ActionLeaveQueue  ActionType = "LEAVE_QUEUE"
	ActionCharge      ActionType = "CHARGE"
	ActionStopCharge  ActionType = "STOP_CHARGE"
)

// what will be send to the environment
type CarAction struct {
	CarID      types.CarID
	Type       ActionType
	TargetNode types.NodeID // Target node ID on the map
}

type Car struct {
	ID          types.CarID
	OriginNode  types.NodeID
	DestNode    types.NodeID
	CurrentNode types.NodeID

	BatteryLevel float64
	MaxBattery   float64
	Consumption  float64

	State           CarState
	TargetStationID types.StationID

	Map environment.Map
	Rng *rand.Rand
}

func NewCar(id types.CarID, orig, dest types.NodeID, battery, maxBattery, consumption float64, envMap environment.Map, rng *rand.Rand) *Car {
	return &Car{
		ID:           id,
		OriginNode:   orig,
		DestNode:     dest,
		CurrentNode:  orig,
		BatteryLevel: battery,
		MaxBattery:   maxBattery,
		Consumption:  consumption,
		State:        StateInRoute,
		Map:          envMap,
		Rng:          rng,
	}
}

func (c *Car) Name() string {
	return fmt.Sprintf("car-%d", c.ID)
}

func (c *Car) Type() types.AgentType {
	return types.AgentTypeCar
}

func (c *Car) Start(ctx context.Context, tickIn <-chan types.SimTime, doneOut chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tickIn:
			action := c.Step(now)
			fmt.Printf("[%s] tick %d received (state: %s, battery: %.1f%%, action: %s)\n",
				c.Name(), now, c.State, c.BatteryLevel*100, action.Type)
			doneOut <- struct{}{}
		}
	}
}

// Step builds the current perception, triggers decision making and returns the chosen action
func (c *Car) Step(now types.SimTime) CarAction {
	percept := CarPerception{
		CurrentTick:  now,
		NodePosition: c.CurrentNode,
		BatteryLevel: c.BatteryLevel,
		StationInfos: c.perceiveStations(),
	}

	return c.Decide(percept)
}

// perceiveStations inspects map stations to build the perception
func (c *Car) perceiveStations() []StationInfoView {
	if c.Map == nil {
		return nil
	}

	stationNodes := c.Map.Stations()
	views := make([]StationInfoView, 0, len(stationNodes))

	for idx, nodeID := range stationNodes {
		views = append(views, StationInfoView{
			StationID:   types.StationID(idx + 1),
			NodeID:      nodeID,
			QueueLength: 0,
			Price:       0,
			Available:   true,
		})
	}

	return views
}

func (c *Car) Range() float64 {
	if c.Consumption <= 0 {
		return 0
	}
	return (c.BatteryLevel * c.MaxBattery) / c.Consumption
}

// Decide determines the next action based on current state and perception
func (c *Car) Decide(percept CarPerception) CarAction {
	var queueLength int

	for _, station := range percept.StationInfos {
		if station.StationID == c.TargetStationID {
			queueLength = station.QueueLength
			break
		}
	}

	switch c.State {
	case StateInRoute:
		_, distDest, _ := c.Map.ShortestPath(c.CurrentNode, c.DestNode)
		if c.BatteryLevel < 0.25 || distDest > c.Range() {
			var bestStation types.StationID
			var bestStationNode types.NodeID
			var minDist float64 = -1

			for _, info := range percept.StationInfos {
				_, dist, err := c.Map.ShortestPath(c.CurrentNode, info.NodeID)
				if err == nil && (minDist < 0 || dist < minDist) {
					minDist = dist
					bestStation = info.StationID
					bestStationNode = info.NodeID
				}
			}

			c.TargetStationID = bestStation
			c.State = StateToStation
			return CarAction{
				CarID:      c.ID,
				Type:       ActionGoToStation,
				TargetNode: bestStationNode,
			}
		}
		return CarAction{
			CarID:      c.ID,
			Type:       ActionMove,
			TargetNode: c.DestNode,
		}

	case StateToStation:
		// Find station node if known, otherwise 0
		var stationNode types.NodeID
		for _, info := range percept.StationInfos {
			if info.StationID == c.TargetStationID {
				stationNode = info.NodeID
				break
			}
		}
		return CarAction{
			CarID:      c.ID,
			Type:       ActionGoToStation,
			TargetNode: stationNode,
		}

	case StateArrivedDestination:
		_, c.DestNode = c.Map.RandomOriginDestination(c.Rng)
		c.State = StateInRoute
		return CarAction{
			CarID:      c.ID,
			Type:       ActionMove,
			TargetNode: c.DestNode,
		}

	case StateAtStation:
		if queueLength > 0 {
			c.State = StateInLine
			return CarAction{
				CarID:      c.ID,
				Type:       ActionEnterQueue,
				TargetNode: 0,
			}
		}
		c.State = StateCharging
		return CarAction{
			CarID:      c.ID,
			Type:       ActionCharge,
			TargetNode: 0,
		}

	case StateInLine:
		if queueLength == 0 {
			c.State = StateCharging
			return CarAction{
				CarID:      c.ID,
				Type:       ActionCharge,
				TargetNode: 0,
			}
		}
		return CarAction{
			CarID:      c.ID,
			Type:       ActionEnterQueue,
			TargetNode: 0,
		}

	case StateCharging:
		if c.BatteryLevel >= 1.0 {
			c.State = StateInRoute
			return CarAction{
				CarID:      c.ID,
				Type:       ActionStopCharge,
				TargetNode: 0,
			}
		}
		return CarAction{
			CarID:      c.ID,
			Type:       ActionCharge,
			TargetNode: 0,
		}

	case StateBrokenDown:
		return CarAction{
			CarID:      c.ID,
			Type:       ActionType("NONE"),
			TargetNode: 0,
		}

	default:
		return CarAction{
			CarID:      c.ID,
			Type:       ActionType("NONE"),
			TargetNode: 0,
		}
	}
}
