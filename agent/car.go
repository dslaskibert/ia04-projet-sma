package agent

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/dslaskibert/ia04-projet-sma/environment"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

var _ Agent = (*Car)(nil)

type CarState string

const (
	StateInRoute    			CarState = "IN_ROUTE"
	StateToStation  			CarState = "TO_STATION"
	StateAtStation				CarState = "AT_STATION"
	StateInLine     			CarState = "IN_LINE"
	StateCharging   			CarState = "CHARGING"
	StateArrivedDestination    	CarState = "ARRIVED_DESTINATION"
	StateBrokenDown 			CarState = "BROKEN_DOWN"
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
	Avaliable   bool
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

// what will be send to the environement
type CarAction struct {
	CarID    types.CarID
	Type     ActionType
	TargetID types.StationID // ID from the next node/station
}

type Car struct {
	ID          	types.CarID
	OriginNode  	types.NodeID //*
	DestNode    	types.NodeID
	CurrentNode 	types.NodeID //*

	BatteryLevel 	float64
	MaxBattery   	float64
	Consumption  	float64

	State           CarState
	TargetStationID types.StationID

	Map             environment.Map
	Rng             *rand.Rand

	In_perception 	chan CarPerception
	Out_Action    	chan CarAction
}

func NewCar(id types.CarID, orig, dest types.NodeID, battery, maxBattery, consumption float64, envMap environment.Map, rng *rand.Rand) *Car {
	return &Car{
		ID:            id,
		OriginNode:    orig,
		DestNode:      dest,
		CurrentNode:   orig,
		BatteryLevel:  battery,
		MaxBattery:    maxBattery,
		Consumption:   consumption,
		State:         StateInRoute,
		Map:           envMap,
		Rng:           rng,
		In_perception: make(chan CarPerception),
		Out_Action:    make(chan CarAction),
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
			fmt.Printf("[%s] tick %d received (state: %s, battery: %.1f%%)\n", c.Name(), now, c.State, c.BatteryLevel*100)
			doneOut <- struct{}{}
		}
	}
}

func (c *Car) Range() float64 {
	if c.Consumption <= 0 {
		return 0
	}
	return (c.BatteryLevel * c.MaxBattery) / c.Consumption
}

// every agent car will have a destination, and it's goal is to reach this destination, when it does, another destination will be given randonly.
func (c *Car) Decide (percept CarPerception) CarAction {

	var QueueLength int

	for _, station := range percept.StationInfos {
		if station.StationID == c.TargetStationID {
			QueueLength = station.QueueLength
			break
		}
	}

	switch c.State {
		case StateInRoute:
			_, distDest, _ := c.Map.ShortestPath(c.CurrentNode, c.DestNode)
			if c.BatteryLevel < 0.25 || distDest > c.Range() {
				var bestStation types.StationID
				var minDist float64 = -1
				for _, info := range percept.StationInfos {
					_, dist, err := c.Map.ShortestPath(c.CurrentNode, info.NodeID)
					if err == nil && (minDist < 0 || dist < minDist) {
						minDist = dist
						bestStation = info.StationID
					}
				}

				c.TargetStationID = bestStation
				c.State = StateToStation
				return CarAction{
					CarID:    c.ID,
					Type:     ActionGoToStation,
					TargetID: c.TargetStationID,
				}
			}
			return CarAction{
				CarID:    c.ID,
				Type:     ActionMove,
				TargetID: types.StationID(c.DestNode),
			}
		case StateToStation:
			return CarAction{
				CarID:    c.ID,
				Type:     ActionGoToStation,
				TargetID: c.TargetStationID,
			}
		case StateArrivedDestination:
			_, c.DestNode = c.Map.RandomOriginDestination(c.Rng)
			c.State = StateInRoute
			return CarAction{
				CarID:    c.ID,
				Type:     ActionMove,
				TargetID: types.StationID(c.DestNode),
			}
		case StateAtStation:
			if QueueLength > 0 {
				c.State = StateInLine

				return CarAction{
					CarID: c.ID,
					Type: ActionEnterQueue,
					TargetID: c.TargetStationID,
				}
			} else {
				c.State = StateCharging

				return CarAction{
					CarID: c.ID,
					Type: ActionCharge,
					TargetID: c.TargetStationID,
				}
			}
		case StateInLine:
			if QueueLength == 0 {
				c.State = StateCharging

				return CarAction{
					CarID: c.ID,
					Type: ActionCharge,
					TargetID: c.TargetStationID,
				}
			} else {
				return CarAction{
					CarID: c.ID,
					Type: ActionEnterQueue,
					TargetID: c.TargetStationID,
				}
			}
		case StateCharging:
			//obs: maybe add some randon (the car leaves before reach 100%)
			if c.BatteryLevel >= 1 {
				c.State = StateInRoute

				return CarAction{
					CarID: c.ID,
					Type: ActionStopCharge,
					TargetID: c.TargetStationID,
				}
			} else {
				return CarAction{
					CarID: c.ID,
					Type: ActionCharge,
					TargetID: c.TargetStationID,
				}
			}
		case StateBrokenDown:
			return CarAction{
				CarID: c.ID,
				Type:  ActionType("NONE"),
			}
		default:
			return CarAction{
				CarID: c.ID,
				Type:  ActionType("NONE"),
			}
	}
}