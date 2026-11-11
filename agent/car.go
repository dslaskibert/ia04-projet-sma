package agent

import (
	"context"

	"github.com/dslaskibert/ia04-projet-sma/types"
)

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

	In_perception 	chan CarPerception
	Out_Action    	chan CarAction
}

func NewCar(id types.CarID, orig, dest types.NodeID, battery, maxBattery, consumption float64) *Car {
	return &Car{
		ID:            id,
		OriginNode:    orig,
		DestNode:      dest,
		CurrentNode:   orig,
		BatteryLevel:  battery,
		MaxBattery:    maxBattery,
		Consumption:   consumption,
		State:         StateInRoute,
		In_perception: make(chan CarPerception),
		Out_Action:    make(chan CarAction),
	}
}

func (c *Car) Start(ctx context.Context) {
	for {
		select {
		case <- ctx.Done():
			return
		case percept, ok := <- c.In_perception:
			if !ok {
				return
			}
			c.CurrentNode = percept.NodePosition
			c.BatteryLevel = percept.BatteryLevel

			action := c.Decide(percept)

			c.Out_Action <- action
		}
	}
}

func (c *Car) Range(batteryLevel float64) float64 {
	if c.Consumption <= 0 {
		return 0
	}
	return (batteryLevel * c.MaxBattery) / c.Consumption
}

func (c *Car) SearchStation(percept CarPerception) types.StationID {
	if len(percept.StationInfos) > 0 {
		return percept.StationInfos[0].StationID
	}
	return 0
}

func (c *Car) NewRandomDestination() types.NodeID {
	// TODO: select the next random destiny
	return c.DestNode
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
			//check the battery
				//if the battery level is lower then 25% or not enouth to arrive at destination
					//search a Station
					// State <- StateToStation
			if (c.BatteryLevel < 0.25 || c.DistDest() > c.Range(c.BatteryLevel)) {
				c.TargetStationID = c.SearchStation(percept)
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
			//change destination *put a temporary destination (the station)
			//State <- StateInRoute
			return CarAction{
				CarID:    c.ID,
				Type:     ActionGoToStation,
				TargetID: c.TargetStationID,
			}
		case StateArrivedDestination:
			//ask for another destination
				//State <- StateInRoute
			c.DestNode = c.NewRandomDestination()
			c.State = StateInRoute
			return CarAction{
				CarID:    c.ID,
				Type:     ActionMove,
				TargetID: types.StationID(c.DestNode),
			}
		case StateAtStation:
			//if there's a queue | QueueLength > 0?
				//wait
					//then charge
			//if there's no queue
				//State <- CHARGING
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
			//wait					
				//State <- CHARGING
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
			//wait	
				//State <- IN_ROUTE
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