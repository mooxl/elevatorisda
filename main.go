package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ControlLogic : The mother of all models
type ControlLogic struct {
	capacityPeople   int
	capacityElevator int
	levels           int
	elevators        int
	runtime          float64
	simulations      int
	function         int
	functions        []interface{}
	building         Building
}

// Building : Defines a building
type Building struct {
	name      string
	people    []Person
	levels    []Level
	elevators []Elevator
	muTex     sync.Mutex
}

// Level : Defines a level in a building
type Level struct {
	identifier    int
	waitingPeople []Person
}

// Elevator : Defines a elevator
type Elevator struct {
	identifier   int
	capacity     int
	distance     int
	currentLevel int
	goingToLevel int
	people       []Person
}

// Person : "Defines" a person
type Person struct {
	waitingOnLevel int
	wantingToLevel int
	timeWaited     int
	timeSpend      int
}

var counter int

func main() {
	central := ControlLogic{}
	central.centralControlLogic()
	central.controlLogic()
	time.Sleep(time.Millisecond)
	central.toString()
}

func (central *ControlLogic) centralControlLogic() {
	central.capacityPeople = 1000
	central.capacityElevator = 5
	central.levels = 6
	central.elevators = 3
	central.runtime = 0.2
	central.simulations = 1
	central.function = 1
	counter = central.capacityPeople
}

func (central *ControlLogic) controlLogic() {
	personChannel := make(chan Person)
	elevatorChannels := make([]chan Elevator, central.elevators)

	central.functions = append(central.functions, LongestWaitingorLongestWanting)
	central.functions = append(central.functions, LongestWaitingorNearestWanting)
	central.building = Building{
		name:      "FH-AACHEN",
		people:    []Person{},
		levels:    []Level{},
		elevators: []Elevator{},
	}
	for i := 0; i < central.elevators; i++ {
		tmpElevator := Elevator{
			identifier: i,
			capacity:   central.capacityElevator,
			people:     []Person{},
		}
		central.building.elevators = append(central.building.elevators, tmpElevator)
	}
	for i := 0; i < central.levels; i++ {
		tmpLevel := Level{
			identifier:    i,
			waitingPeople: []Person{},
		}
		central.building.levels = append(central.building.levels, tmpLevel)
	}

	central.elevator(elevatorChannels)
	go central.person(personChannel)

	for counter != 0 {
		select {
		case person := <-personChannel:
			central.building.levels[person.waitingOnLevel].waitingPeople = append(central.building.levels[person.waitingOnLevel].waitingPeople, person)
			fmt.Println("new person waiting on Floor", person.waitingOnLevel, "who wants to", person.wantingToLevel)
		default:
			for _, elevatorChannel := range elevatorChannels {
				select {
				case elevator := <-elevatorChannel:
					central.functions[central.function].(func(*Elevator, *ControlLogic))(&elevator, central)
				default:
					continue
				}
			}
		}
	}

}

func (central *ControlLogic) person(personChannel chan<- Person) {
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < central.capacityPeople; i++ {
		go func() {
			person := Person{
				timeWaited:     1,
				waitingOnLevel: rand.Intn(len(central.building.levels)),
			}
			for {
				person.wantingToLevel = rand.Intn(len(central.building.levels))
				if person.waitingOnLevel == person.wantingToLevel {
					continue
				} else {
					break
				}
			}
			personChannel <- person
		}()
	}
}

func (central *ControlLogic) elevator(elevatorChannels []chan Elevator) {
	for i := range elevatorChannels {
		ii := i
		go func(central *ControlLogic) {
			elevatorChannels[ii] = make(chan Elevator)
			elevator := &central.building.elevators[ii]
			people := &central.building.elevators[ii].people
			counterForTime := 0
			for {
				counterForTime++
				tempElevatorPeople := []Person{}
				if len(*people) != 0 {
					for _, person := range *people {
						person.timeSpend++
						if elevator.currentLevel == person.wantingToLevel {
							fmt.Println("person who wanted to level", person.wantingToLevel, "left the elevator #", elevator.identifier, "and waited altogether", person.timeWaited+person.timeSpend)
							central.building.people = append(central.building.people, person)
							counter--
						} else {
							tempElevatorPeople = append(tempElevatorPeople, person)
						}
					}
					*people = tempElevatorPeople
				}
				for len(*people) < elevator.capacity {
					central.building.muTex.Lock()
					if len(central.building.levels[elevator.currentLevel].waitingPeople) != 0 {
						longestWaiter := central.building.levels[elevator.currentLevel].waitingPeople[0]
						central.building.levels[elevator.currentLevel].waitingPeople = central.building.levels[elevator.currentLevel].waitingPeople[1:]
						longestWaiter.timeWaited = counterForTime
						*people = append(*people, longestWaiter)
						fmt.Println("person who wants to", longestWaiter.wantingToLevel, "went in the elevator #", elevator.identifier)
						central.building.muTex.Unlock()
					} else {
						central.building.muTex.Unlock()
						break
					}
				}
				if elevator.currentLevel < elevator.goingToLevel {
					elevator.distance++
					elevator.currentLevel++
					fmt.Println("Elevator #", elevator.identifier, "is now on", elevator.currentLevel)
				} else if elevator.currentLevel > elevator.goingToLevel {
					elevator.currentLevel--
					elevator.distance++
					fmt.Println("Elevator #", elevator.identifier, "is now on", elevator.currentLevel)
				} else {
					elevatorChannels[ii] <- *elevator
				}
			}
		}(central)
	}
}

// LongestWaitingorLongestWanting : If the elevator is empty, it goes to the level, which has the longest waiting person
//									If the elevator has people inside, it goes to the level, which is the destination of the longest waiting person in the elevator
func LongestWaitingorLongestWanting(elevator *Elevator, central *ControlLogic) {
	longestWaitingorLongestWantingPerson := Person{}
	if len(elevator.people) == 0 {
		for _, level := range central.building.levels {
			for _, person := range level.waitingPeople {
				if person.timeWaited > longestWaitingorLongestWantingPerson.timeWaited {
					longestWaitingorLongestWantingPerson = person
				}
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorLongestWantingPerson.waitingOnLevel
	} else {
		for _, person := range elevator.people {
			if person.timeSpend >= longestWaitingorLongestWantingPerson.timeSpend {
				longestWaitingorLongestWantingPerson = person
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorLongestWantingPerson.wantingToLevel
	}

	if elevator.currentLevel != central.building.elevators[elevator.identifier].goingToLevel {
		fmt.Println("Elevator #", elevator.identifier, "is now on level", elevator.currentLevel, "and is going to level", central.building.elevators[elevator.identifier].goingToLevel)
	}

}

// LongestWaitingorNearestWanting : If the elevator is empty, it goes to the level, which has the longest waiting person
//									If the elevator has people inside, it goes to the level, which is the nearest destination of the waiting people in the elevator
func LongestWaitingorNearestWanting(elevator *Elevator, central *ControlLogic) {
	longestWaitingorNearestWantingPerson := Person{}
	if len(elevator.people) == 0 {
		for _, level := range central.building.levels {
			for _, person := range level.waitingPeople {
				if person.timeWaited > longestWaitingorNearestWantingPerson.timeWaited {
					longestWaitingorNearestWantingPerson = person
				}
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorNearestWantingPerson.waitingOnLevel
	} else {
		tmpCurrent := elevator.currentLevel
		for i, person := range elevator.people {
			if i == 0 {
				longestWaitingorNearestWantingPerson = person
			} else {
				if person.wantingToLevel > tmpCurrent {
					if longestWaitingorNearestWantingPerson.wantingToLevel > person.wantingToLevel {
						longestWaitingorNearestWantingPerson = person
					}
				} else {
					if person.wantingToLevel > longestWaitingorNearestWantingPerson.wantingToLevel {
						longestWaitingorNearestWantingPerson = person
					}
				}
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorNearestWantingPerson.wantingToLevel
	}

	if elevator.currentLevel != central.building.elevators[elevator.identifier].goingToLevel {
		fmt.Println("Elevator #", elevator.identifier, "is now on level", elevator.currentLevel, "and is going to level", central.building.elevators[elevator.identifier].goingToLevel)
	}
}

func (central *ControlLogic) toString() {
	fmt.Println()
	fmt.Println("Name:\t", central.building.name)
	fmt.Println("Capacity:\t", central.capacityPeople)
	fmt.Println()
	/* fmt.Println("Levels:")
	for _, level := range central.building.levels {
		fmt.Println("\tIdentifier:\t", level.identifier)
		fmt.Println("\tWaiting people:\t", level.waitingPeople)
		fmt.Println()
	} */
	/* fmt.Println("Elevators:")
	for _, elevator := range central.building.elevators {
		fmt.Println("\tIdentifier:\t", elevator.identifier)
		fmt.Println("\tCapacity:\t", elevator.capacity)
		fmt.Println("\tDistance:\t", elevator.distance)
		fmt.Println("\tPeople:\t", elevator.people)
		fmt.Println()
	} */
	absoluteWaitTime := 0
	absoluteSpendTime := 0
	for _, person := range central.building.people {
		absoluteWaitTime += person.timeWaited
		absoluteSpendTime += person.timeSpend
	}
	fmt.Println("Average waiting time:", absoluteWaitTime/central.capacityPeople)
	fmt.Println("Average spending time:", absoluteSpendTime/central.capacityPeople)
	/* fmt.Println("People:")
	for _, human := range central.building.people {
		fmt.Println("\tTime waited:\t", human.timeWaited)
		fmt.Println("\tTime spend:\t", human.timeSpend)
		fmt.Println()
	} */
}
