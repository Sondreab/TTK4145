package main

import (
	"./config"
	"./driver"
	//"./fsm"
	"./network"
	"./network/localip"
	"fmt"
	"time"
)

func initializeLiftData() config.Lift {
	var lift config.Lift
	id, err := localip.LocalIP()
	if err == nil {
		lift = config.Lift{
			ID: id,
			Alive: true,
			LastKnownFloor: -1,
			TargetFloor: -1,
			MotorDir: config.MD_Stop,
			Behaviour: config.LiftIdle}

	}

	return lift
}

var message config.Message
var localLift config.LiftUpdate
var outboundMap config.NodeMap

func main() {

	thisLift := initializeLiftData()
	myID := thisLift.ID

	/*//##### FSM Init #####
	LiftToFsmCh :=make(chan config.Lift)

	LiftFromFsmCh :=make(chan config.Lift)
	if driver.GetFloorSensorSignal() == -1 {
		fsm.FsmOnInitBetweenFloors(LiftToFsmCh)
	}

	go fsm.FsmLoop(LiftToFsmCh,LiftFromFsmCh)
	//send to FSM
	LiftToFsmCh <- ThisLift
	//recieve from FSM
	someLift := <- LiftFromFsmCh
	*/

	//Initialize maps like this:
	var NodeMap config.NodeMap
	NodeMap = make(config.NodeMap)
	NodeMap[ThisLift.ID] = thisLift

	send := make(chan config.Message)
	recieve := make(chan config.Message)
	lostPeers := make(chan []string)

	//compiler channels
	recievedMsg := make(chan config.Message)
	sendMap := make(chan config.NodeMap)
	liftToCompiler := make(chan config.LiftUpdate)
	disconnectedNodes := make(chan []string)
	

	//polling channels
	polledButton := make(chan config.ButtonEvent)
	polledFloorSensor := make(chan int)

	//FSM channels
	liftToFsm := make(chan config.Lift)
	liftFromFsm := make(chan config.Lift)

	//COST channels
	mapToCost := make(chan config.NodeMap)
	liftFromCost := make(chan config.Lift)

	//Starting threads
	go network.Network(send, recieve, lostPeers)
	go nodeMapCompiler(recievedMsg, sendMap, liftToCompiler, disconnectedNodes)
	go driver.PollButtons(polledButton)
	go driver.PollFloorSensor(polledFloorSensor)



	go func() {
		test := config.Message{NodeMap, ThisLift.ID, 0}
		for {
			send <- test
			test.Iter++
			time.Sleep(1 * time.Second)
			fmt.Println("sending")
		}
	}()

	for {
		select {
		case p := <-lostPeers:
			fmt.Println(p)
		case r := <-recieve:
			fmt.Println("recieved: ", r)
		}
	}


	//Scetch of main loop
	for{
		select{
			case p := <-lostPeers:
				disconnectedNodes <- p 

			case message := <- recieve:
				recievedMsg <- message

			case outboundMap := <- sendMap:
				thisLift = outboundMap[myID]
				message.ID = myID
				message.NodeMap = outboundMap
				send <- message
				mapToCost <- outboundMap
				liftToFsm <- thisLift
				
			case button := <- polledButton:
				liftData := initializeLiftData()
				liftData.Requests[button.Floor][button.Button] = true
				localLift.Lift = liftData
				localLift.Source = config.Button_Poll
				liftToCompiler <- localLift

			case floor := <- polledFloorSensor:
				liftData := initializeLiftData()
				liftData.LastKnownFloor = floor
				localLift.Lift = liftData
				localLift.Source = config.Floor_Poll
				liftToCompiler <- localLift

			case liftData := <- liftFromFsm:
				localLift.Lift = liftData
				localLift.Source = config.FSM
				liftToCompiler <- localLift

			case liftData := <- liftFromCost
				localLift.Lift = liftData
				localLift.Source = config.Cost
				liftToCompiler <- localLift
		}

	}
}
