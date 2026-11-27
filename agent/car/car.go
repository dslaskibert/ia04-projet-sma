package car

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/agent"
	"github.com/dslaskibert/ia04-projet-sma/environment"
	"github.com/dslaskibert/ia04-projet-sma/messages"
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
	id          		types.CarID
	originNode  		types.NodeID
	destNode    		types.NodeID
	currentNode 		types.NodeID
	path        		[]types.NodeID

	batteryLevel 		float64
	maxBattery   		float64
	consumption  		float64

	state           	CarState
	targetStationID 	types.StationID

	travelTimeMin   int     // minutes runing
	waitingTimeMin  int     // minutes in the line
	chargingTimeMin int     // minutes charging
	distanceKm      float64 // total range in km
	stationChanges  int     // changes of target stations

	Map 				environment.Map
	Rng 				*rand.Rand
	registry 			types.Registry
	replyChan			chan messages.StationReply
}

type CarView struct {
	ID              types.CarID
	CurrentNode     types.NodeID
	DestNode        types.NodeID
	BatteryLevel    float64
	MaxBattery      float64
	State           CarState
	TargetStationID types.StationID

	TravelTimeMin   int
	WaitingTimeMin  int
	ChargingTimeMin int
	DistanceKm      float64
	StationChanges  int
}

func (c *Car) Snapshot() CarView {
	return CarView{
		ID:              c.id,
		CurrentNode:     c.currentNode,
		DestNode:        c.destNode,
		BatteryLevel:    c.batteryLevel,
		MaxBattery:      c.maxBattery,
		State:           c.state,
		TargetStationID: c.targetStationID,
		TravelTimeMin:   c.travelTimeMin,
		WaitingTimeMin:  c.waitingTimeMin,
		ChargingTimeMin: c.chargingTimeMin,
		DistanceKm:      c.distanceKm,
		StationChanges:  c.stationChanges,
	}
}

func (c *Car) ID() types.CarID {
	return c.id
}

func (c *Car) Position() types.NodeID {
	return c.currentNode
}

func NewCar(id types.CarID, orig, dest types.NodeID, battery, maxBattery, consumption float64, envMap environment.Map, rng *rand.Rand) *Car {
	return &Car{
		id:           id,
		originNode:   orig,
		destNode:     dest,
		currentNode:  orig,
		batteryLevel: battery,
		maxBattery:   maxBattery,
		consumption:  consumption,
		state:        StateInRoute,
		Map:          envMap,
		Rng:          rng,
		replyChan: 	  make(chan messages.StationReply, 10),
	}
}

func (c *Car) SetRegistry(reg types.Registry) {
	c.registry = reg
}

func (c *Car) Name() string {
	return fmt.Sprintf("car-%d", c.id)
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
			fmt.Printf("[%s] tick %d received (pos: %d, dest: %d, state: %s, battery: %.1f%%, action: %s)\n",
				c.Name(), now, c.currentNode, c.destNode, c.state, c.batteryLevel*100, action.Type)

			doneOut <- struct{}{}
		}
	}
}

// Step builds the current perception, triggers decision making and returns the chosen action
func (c *Car) Step(now types.SimTime) CarAction {
	c.receiveReplies()

	switch c.state {
	case StateInLine:
		c.waitingTimeMin++
	case StateCharging:
		c.chargingTimeMin++
	}

	percept := CarPerception{
		CurrentTick:  now,
		NodePosition: c.currentNode,
		BatteryLevel: c.batteryLevel,
		StationInfos: c.perceiveStations(),
	}

	action := c.Decide(percept)

	if action.Type == ActionMove || action.Type == ActionGoToStation {
		c.moveNextNode()
	}

	return action
}

func (c *Car) moveNextNode() {
	if c.Map == nil || len(c.path) < 2 {
		return
	}

	nextNode := c.path[1]
	var dist float64
	var edgeFound bool
	for _, edge := range c.Map.Edges(c.currentNode) {
		if edge.To() == nextNode {
			dist = edge.Length()
			edgeFound = true
			break
		}
	}
	if !edgeFound {
		return
	}

	c.currentNode = nextNode
	c.path = c.path[1:]

	c.travelTimeMin++
	c.distanceKm += dist / 1000

	c.batteryLevel -= (dist * c.consumption) / (c.maxBattery * 1000)
	if c.batteryLevel <= 0 {
		c.batteryLevel = 0
		c.state = StateBrokenDown
		return
	}

	// Check arrival at destination
	if c.state == StateInRoute && c.currentNode == c.destNode {
		c.state = StateArrivedDestination
		c.path = nil
	}
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
	if c.consumption <= 0 {
		return 0
	}
	return (c.batteryLevel * c.maxBattery / c.consumption) * 1000
}

// Decide determines the next action based on current state and perception
func (c *Car) Decide(percept CarPerception) CarAction {
	var queueLength int

	for _, station := range percept.StationInfos {
		if station.StationID == c.targetStationID {
			queueLength = station.QueueLength
			break
		}
	}

	switch c.state {
	case StateInRoute:
		_, distDest, _ := c.Map.ShortestPath(c.currentNode, c.destNode)
		if c.batteryLevel < 0.25 || distDest > c.Range() {
			var bestStation types.StationID
			var bestStationNode types.NodeID
			var bestPath []types.NodeID
			var minDist float64 = -1
			
			for _, info := range percept.StationInfos {
				pathSt, dist, err := c.Map.ShortestPath(c.currentNode, info.NodeID)
				if err == nil && (minDist < 0 || dist < minDist) {
					minDist = dist
					bestStation = info.StationID
					bestStationNode = info.NodeID
					bestPath = pathSt
				}
			}

			if c.targetStationID != 0 && c.targetStationID != bestStation {
				c.stationChanges++
			}

			c.targetStationID = bestStation
			c.state = StateToStation
			c.path = bestPath
			return CarAction{
				CarID:      c.id,
				Type:       ActionGoToStation,
				TargetNode: bestStationNode,
			}
		}

		if len(c.path) < 2 {
			c.path, _, _ = c.Map.ShortestPath(c.currentNode, c.destNode)
		}

		return CarAction{
			CarID:      c.id,
			Type:       ActionMove,
			TargetNode: c.destNode,
		}

	case StateToStation:
		// Find station node if known, otherwise 0
		var stationNode types.NodeID
		for _, info := range percept.StationInfos {
			if info.StationID == c.targetStationID {
				stationNode = info.NodeID
				break
			}
		}

		if c.currentNode == stationNode {
			c.state = StateAtStation
			c.path = nil
			return CarAction{
				CarID:      c.id,
				Type:       ActionType("NONE"),
				TargetNode: stationNode,
			}
		}

		if len(c.path) < 2 {
			c.path, _, _ = c.Map.ShortestPath(c.currentNode, stationNode)
		}

		return CarAction{
			CarID:      c.id,
			Type:       ActionGoToStation,
			TargetNode: stationNode,
		}

	case StateArrivedDestination:
		_, c.destNode = c.Map.RandomOriginDestination(c.Rng)
		c.state = StateInRoute
		c.path, _, _ = c.Map.ShortestPath(c.currentNode, c.destNode)
		return CarAction{
			CarID:      c.id,
			Type:       ActionMove,
			TargetNode: c.destNode,
		}

	case StateAtStation:
		if c.registry != nil {
			st := c.registry.StationByID(c.targetStationID)
			st.Inbox() <- messages.ArriveMessage{
				CarID:     c.id,
				NeededKWh: (1.0 - c.batteryLevel) * c.maxBattery,
				ReplyChan: c.replyChan,
			}
			c.state = StateInLine
			return CarAction{
				CarID:      c.id,
				Type:       ActionEnterQueue,
				TargetNode: 0,
			}
		}

		if queueLength > 0 {
			c.state = StateInLine
			return CarAction{
				CarID:      c.id,
				Type:       ActionEnterQueue,
				TargetNode: 0,
			}
		}
		c.state = StateCharging
		return CarAction{
			CarID:      c.id,
			Type:       ActionCharge,
			TargetNode: 0,
		}

	case StateInLine:
		if queueLength == 0 {
			c.state = StateCharging
			return CarAction{
				CarID:      c.id,
				Type:       ActionCharge,
				TargetNode: 0,
			}
		}
		return CarAction{
			CarID:      c.id,
			Type:       ActionEnterQueue,
			TargetNode: 0,
		}

		case StateCharging:
		if c.registry != nil {
			return CarAction{
				CarID:      c.id,
				Type:       ActionCharge,
				TargetNode: 0,
			}
		}
		if c.batteryLevel >= 1.0 {
			c.state = StateInRoute
			return CarAction{
				CarID:      c.id,
				Type:       ActionStopCharge,
				TargetNode: 0,
			}
		}
		return CarAction{
			CarID:      c.id,
			Type:       ActionCharge,
			TargetNode: 0,
		}

	case StateBrokenDown:
		return CarAction{
			CarID:      c.id,
			Type:       ActionType("NONE"),
			TargetNode: 0,
		}

	default:
		return CarAction{
			CarID:      c.id,
			Type:       ActionType("NONE"),
			TargetNode: 0,
		}
	}
}

func (c *Car) receiveReplies() {
	if c.replyChan == nil {
		return
	}
	for {
		select {
		case rep := <- c.replyChan:
			switch m := rep.(type) {
			case messages.QueuePositionMessage:
				if m.Position == 0 {
					c.state = StateCharging
				}
			case messages.ChargeFinishedMessage:
				c.batteryLevel += m.EnergyGiven / c.maxBattery
				if c.batteryLevel > 1.0 {
					c.batteryLevel = 1.0
				}
				c.state = StateInRoute
				c.path = nil
				if c.Map != nil && c.Rng != nil {
					_, c.destNode = c.Map.RandomOriginDestination(c.Rng)
					c.path, _, _ = c.Map.ShortestPath(c.currentNode, c.destNode)
				}
				c.targetStationID = 0
			}
		default:
			return
		}
	}
}