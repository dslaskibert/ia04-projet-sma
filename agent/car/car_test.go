package car

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
	gridMap := gridmap.NewGridMap(5, 5, 100.0, [][2]int{{2, 2}})
	orig, dest := gridMap.RandomOriginDestination(rng)

	fmt.Printf("[INIT] GridMap created: 5x5 grid (25 nodes, 100m edges)\n")
	fmt.Printf("[INIT] Charging station configured at node 12 (position (2,2))\n")
	fmt.Printf("[INIT] RandomOriginDestination called -> Origin Node: %d, Destination Node: %d\n", orig, dest)

	car := NewCar(1, orig, dest, 0.8, 100.0, 0.2, gridMap, rng)
	fmt.Printf("[INIT] Car created -> ID: %d, Origin: %d, Destination: %d, Battery: %.1f%%, Autonomy Range: %.1fm\n",
		car.id, car.originNode, car.destNode, car.batteryLevel*100, car.Range())
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
		path, dist, err := gridMap.ShortestPath(car.currentNode, car.destNode)
		fmt.Printf("  -> Dijkstra call: ShortestPath(CurrentNode=%d, DestNode=%d)\n", car.currentNode, car.destNode)
		fmt.Printf("     Path: %v, Total Distance: %.1fm, Error: %v\n", path, dist, err)
		fmt.Printf("  -> Car checks battery range: Range(80%%) = %.1fm (Distance needed = %.1fm)\n", car.Range(), dist)

		percept := CarPerception{
			CurrentTick:  1,
			NodePosition: car.currentNode,
			BatteryLevel: 0.8,
			StationInfos: []StationInfoView{
				{StationID: 101, NodeID: 12, QueueLength: 1, Available: true},
			},
		}
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action := car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetNode = %d, State = %s\n", action.Type, action.TargetNode, car.state)
		if action.Type != ActionMove || car.state != StateInRoute || action.TargetNode != car.destNode {
			t.Errorf("expected ActionMove towards DestNode=%d in StateInRoute, got action=%s target=%d state=%s",
				car.destNode, action.Type, action.TargetNode, car.state)
		}

		// Scenario B: Low battery (< 25%) -> Finds nearest station using Dijkstra
		fmt.Println("\n  [SCENARIO B: Battery drops to 15% (< 25% threshold) - Rerouting to station]")
		car.batteryLevel = 0.15
		percept.BatteryLevel = 0.15
		pathSt, distSt, errSt := gridMap.ShortestPath(car.currentNode, 12)
		fmt.Printf("  -> Dijkstra call: ShortestPath(CurrentNode=%d, StationNode=12)\n", car.currentNode)
		fmt.Printf("     Path to Station: %v, Distance to Station: %.1fm, Error: %v\n", pathSt, distSt, errSt)

		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetNode = %d, State = %s\n", action.Type, action.TargetNode, car.state)
		if action.Type != ActionGoToStation || car.state != StateToStation || action.TargetNode != 12 {
			t.Errorf("expected ActionGoToStation to station node 12, got action=%s target=%d", action.Type, action.TargetNode)
		}

		// Scenario C: Arrived at station with queue
		fmt.Println("\n  [SCENARIO C: Car arrives at Station 101, perception reveals QueueLength = 1]")
		car.state = StateAtStation
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetNode = %d, New State = %s\n", action.Type, action.TargetNode, car.state)
		if action.Type != ActionEnterQueue || car.state != StateInLine || action.TargetNode != 0 {
			t.Errorf("expected ActionEnterQueue with TargetNode=0 in StateInLine, got action=%s target=%d state=%s", action.Type, action.TargetNode, car.state)
		}

		// Scenario D: Queue is cleared
		fmt.Println("\n  [SCENARIO D: Car in line, perception reveals QueueLength = 0 (free charger)]")
		percept.StationInfos[0].QueueLength = 0
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetNode = %d, New State = %s\n", action.Type, action.TargetNode, car.state)
		if action.Type != ActionCharge || car.state != StateCharging || action.TargetNode != 0 {
			t.Errorf("expected ActionCharge with TargetNode=0 in StateCharging, got action=%s target=%d state=%s", action.Type, action.TargetNode, car.state)
		}

		// Scenario E: Fully charged (100%)
		fmt.Println("\n  [SCENARIO E: Battery level reaches 100% (1.0)]")
		car.batteryLevel = 1.0
		percept.BatteryLevel = 1.0
		fmt.Printf("  -> Calling car.Decide(perception)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, TargetNode = %d, New State = %s\n", action.Type, action.TargetNode, car.state)
		if action.Type != ActionStopCharge || car.state != StateInRoute || action.TargetNode != 0 {
			t.Errorf("expected ActionStopCharge with TargetNode=0 in StateInRoute, got action=%s target=%d state=%s", action.Type, action.TargetNode, car.state)
		}

		// Scenario F: Destination reached -> draws new destination on the map
		fmt.Println("\n  [SCENARIO F: Car reaches Destination -> draws new random destination]")
		previousDest := car.destNode
		car.state = StateArrivedDestination
		fmt.Printf("  -> Previous Destination Node: %d\n", previousDest)
		fmt.Printf("  -> Calling car.Decide(perception) (which invokes gridMap.RandomOriginDestination)...\n")
		action = car.Decide(percept)
		fmt.Printf("  -> Decision Result: ActionType = %s, New Destination Node = %d, New State = %s\n",
			action.Type, car.destNode, car.state)
		if action.Type != ActionMove || car.state != StateInRoute || action.TargetNode != car.destNode {
			t.Errorf("expected ActionMove towards new DestNode=%d in StateInRoute, got action=%s target=%d state=%s",
				car.destNode, action.Type, action.TargetNode, car.state)
		}
		fmt.Println("--------------------------------------------------------------------------------")
	
	// Subtest 4: Snapshot
	t.Run("Snapshot", func(t *testing.T) {
		snap := car.Snapshot()

		fmt.Printf("\n[SUBTEST 4: SNAPSHOT]\n")
		fmt.Printf("	-> Snapshot captured: %+v\n", snap)

		if snap.ID != car.id {
			t.Errorf("expected ID %d, got %d", car.id, snap.ID)
		}
		if snap.BatteryLevel != car.batteryLevel {
			t.Errorf("expected BatteryLevel %f, got %f", snap.BatteryLevel, car.batteryLevel)
		}
		if snap.MaxBattery != car.maxBattery {
			t.Errorf("expected MaxBattery %f, got %f", snap.MaxBattery, car.maxBattery)
		}
		if snap.CurrentNode != car.currentNode {
			t.Errorf("expected CurrentNode %d, got %d", snap.CurrentNode, car.currentNode)
		}
		if snap.DestNode != car.destNode {
			t.Errorf("expected DestNode %d, got %d", snap.DestNode, car.destNode)
		}
		if snap.State != car.state {
			t.Errorf("expected State %s, got %s", snap.State, car.state)
		}
		if snap.TargetStationID != car.targetStationID {
			t.Errorf("expected TargetStationID %d, got %d", snap.TargetStationID, car.targetStationID)
		}
		fmt.Printf("	-> Result: PASS\n")
		}) 

	})
	fmt.Println("================================================================================")
}
