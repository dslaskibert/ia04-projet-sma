package agent

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/dslaskibert/ia04-projet-sma/environment/gridmap"
	"github.com/dslaskibert/ia04-projet-sma/types"
)

func TestCar(t *testing.T) {
	fmt.Println("\n================================================================================")
	fmt.Println("                   CAR AGENT COMPREHENSIVE TEST SUITE                           ")
	fmt.Println("================================================================================")

	rng := rand.New(rand.NewSource(42))

	// Create 5x5 GridMap with 100m edges and station at node (2,2) -> ID 12
	GridMap := gridmap.NewGridMap(5, 5, 100.0, [][2]int{{2, 2}})
	orig, dest := GridMap.RandomOriginDestination(rng)

	fmt.Printf("[INIT] GridMap created: 5x5 grid (25 nodes, 100m edges)\n")
	fmt.Printf("[INIT] Charging station configured at node 12 (position (2,2))\n")
	fmt.Printf("[INIT] RandomOriginDestination called -> Origin Node: %d, Destination Node: %d\n", orig, dest)

	car := NewCar(1, orig, dest, 0.8, 100.0, 0.2, GridMap, rng)
	fmt.Printf("[INIT] Car created -> ID: %d, Origin: %d, Destination: %d, Battery: %.1f%%, Autonomy Range: %.1fm\n",
		car.ID, car.OriginNode, car.DestNode, car.BatteryLevel*100, car.Range())
	fmt.Println("--------------------------------------------------------------------------------")

	// Subtest 1: Agent interface check
	t.Run("Agent_Interface", func(t *testing.T) {
		fmt.Printf("\n[SUBTEST 1: AGENT INTERFACE]\n")
		fmt.Printf("  -> Calling car.Name(): '%s'\n", car.Name())
		fmt.Printf("  -> Calling car.Type(): '%s'\n", car.Type())

		if car.Name() != "car-1" {
			t.Errorf("expected Name() = car-1, got %s", car.Name())
		}
		if car.Type() != types.AgentTypeCar {
			t.Errorf("expected Type() = %s, got %s", types.AgentTypeCar, car.Type())
		}
		fmt.Printf("  -> Result: PASS (Interface Agent correctly implemented)\n")
	})

	// Subtest 2: Simulation tick lifecycle
	t.Run("Simulation_Lifecycle", func(t *testing.T) {
		fmt.Printf("\n[SUBTEST 2: SIMULATION LIFECYCLE (TICKS & CHANNELS)]\n")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tickIn := make(chan types.SimTime)
		doneOut := make(chan struct{})

		fmt.Printf("  -> Spawning goroutine: go car.Start(ctx, tickIn, doneOut)\n")
		go car.Start(ctx, tickIn, doneOut)

		for tick := 1; tick <= 3; tick++ {
			fmt.Printf("  -> Simulator sends tick %d through tickIn...\n", tick)
			tickIn <- types.SimTime(tick)
			select {
			case <-doneOut:
				fmt.Printf("  -> Simulator received acknowledgment from doneOut for tick %d\n", tick)
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("timeout waiting for doneOut at tick %d", tick)
			}
		}
		fmt.Printf("  -> Result: PASS (Goroutine handled all ticks synchronously)\n")
	})

	// Subtest 3: Decisions with Dijkstra path calculations
	t.Run("Decisions_With_GridMap", func(t *testing.T) {
		fmt.Printf("\n[SUBTEST 3: DECISION ENGINE (STATE MACHINE & DIJKSTRA CALLS)]\n")

		// Scenario A: Normal travel with high battery
		fmt.Println("\n  [SCENARIO A: Normal travel in route with 80% battery]")
		path, dist, err := GridMap.ShortestPath(car.CurrentNode, car.DestNode)
		fmt.Printf("  -> Dijkstra call: ShortestPath(CurrentNode=%d, DestNode=%d)\n", car.CurrentNode, car.DestNode)
		fmt.Printf("     Path: %v, Total Distance: %.1fm, Error: %v\n", path, dist, err)
		fmt.Printf("  -> Car checks battery range: Range(80%%) = %.1fm (Distance needed = %.1fm)\n", car.Range(), dist)

		percept := CarPerception{
			CurrentTick:  1,
			NodePosition: car.CurrentNode,
			BatteryLevel: 0.8,
			StationInfos: []StationInfoView{
				{StationID: 101, NodeID: 12, QueueLength: 1},
			},
		}
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action := car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetID = %d, State = %s\n", action.Type, action.TargetID, car.State)
		if action.Type != ActionMove || car.State != StateInRoute {
			t.Errorf("expected ActionMove in StateInRoute, got action=%s state=%s", action.Type, car.State)
		}

		// Scenario B: Low battery (< 25%) -> Finds nearest station using Dijkstra
		fmt.Println("\n  [SCENARIO B: Battery drops to 15% (< 25% threshold) - Rerouting to station]")
		car.BatteryLevel = 0.15
		percept.BatteryLevel = 0.15
		pathSt, distSt, errSt := GridMap.ShortestPath(car.CurrentNode, 12)
		fmt.Printf("  -> Dijkstra call: ShortestPath(CurrentNode=%d, StationNode=12)\n", car.CurrentNode)
		fmt.Printf("     Path to Station: %v, Distance to Station: %.1fm, Error: %v\n", pathSt, distSt, errSt)

		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetStationID = %d, State = %s\n", action.Type, action.TargetID, car.State)
		if action.Type != ActionGoToStation || car.State != StateToStation || car.TargetStationID != 101 {
			t.Errorf("expected ActionGoToStation to station 101, got action=%s target=%d", action.Type, car.TargetStationID)
		}

		// Scenario C: Arrived at station with queue
		fmt.Println("\n  [SCENARIO C: Car arrives at Station 101, perception reveals QueueLength = 1]")
		car.State = StateAtStation
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetStationID = %d, New State = %s\n", action.Type, action.TargetID, car.State)
		if action.Type != ActionEnterQueue || car.State != StateInLine {
			t.Errorf("expected ActionEnterQueue in StateInLine, got action=%s state=%s", action.Type, car.State)
		}

		// Scenario D: Queue is cleared
		fmt.Println("\n  [SCENARIO D: Car in line, perception reveals QueueLength = 0 (free charger)]")
		percept.StationInfos[0].QueueLength = 0
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetStationID = %d, New State = %s\n", action.Type, action.TargetID, car.State)
		if action.Type != ActionCharge || car.State != StateCharging {
			t.Errorf("expected ActionCharge in StateCharging, got action=%s state=%s", action.Type, car.State)
		}

		// Scenario E: Fully charged (100%)
		fmt.Println("\n  [SCENARIO E: Battery level reaches 100% (1.0)]")
		car.BatteryLevel = 1.0
		percept.BatteryLevel = 1.0
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetStationID = %d, New State = %s\n", action.Type, action.TargetID, car.State)
		if action.Type != ActionStopCharge || car.State != StateInRoute {
			t.Errorf("expected ActionStopCharge in StateInRoute, got action=%s state=%s", action.Type, car.State)
		}

		// Scenario F: Destination reached -> draws new destination on the map
		fmt.Println("\n  [SCENARIO F: Car reaches Destination -> draws new random destination]")
		previousDest := car.DestNode
		car.State = StateArrivedDestination
		fmt.Printf("  -> Previous Destination Node: %d\n", previousDest)
		fmt.Printf("  -> Calling car.Decide(perception) (which invokes GridMap.RandomOriginDestination)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, New Destination Node = %d, New State = %s\n",
			action.Type, car.DestNode, car.State)
		if action.Type != ActionMove || car.State != StateInRoute {
			t.Errorf("expected ActionMove in StateInRoute, got action=%s state=%s", action.Type, car.State)
		}
		fmt.Println("--------------------------------------------------------------------------------")
	})
	fmt.Println("================================================================================")
}
