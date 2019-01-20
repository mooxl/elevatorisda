package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

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

type Building struct {
	name      string
	people    []Person
	levels    []Level
	elevators []Elevator
	muTex     sync.Mutex
}

type Level struct {
	identifier    int
	waitingPeople []Person
}

type Elevator struct {
	identifier   int
	capacity     int
	distance     int
	currentLevel int
	goingToLevel int
	people       []Person
}

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
}

func (central *ControlLogic) centralControlLogic() {
	central.capacityPeople = 30
	central.capacityElevator = 5
	central.levels = 4
	central.elevators = 2
	central.runtime = 0.2
	central.simulations = 1
	central.function = 0

	counter = central.capacityPeople
}

func (central *ControlLogic) controlLogic() {
	personChannel := make(chan Person)
	elevatorChannels := make([]chan Elevator, central.elevators)

	central.functions = append(central.functions, LongestWaitingorWanting)
	central.functions = append(central.functions, LongestWanting)
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
			central.building.people = append(central.building.people, person)
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
			for {
				elevator := &central.building.elevators[ii]
				tempElevatorPeople := []Person{}
				if len(elevator.people) != 0 {
					for _, person := range elevator.people {
						if elevator.currentLevel == person.wantingToLevel {
							fmt.Println("person who wanted to level", person.wantingToLevel, "left the elevator #", elevator.identifier)
							counter--
						} else {
							person.timeSpend = 1
							tempElevatorPeople = append(tempElevatorPeople, person)
						}
					}
					elevator.people = tempElevatorPeople
				}

				for len(elevator.people) < elevator.capacity {
					central.building.muTex.Lock()
					if len(central.building.levels[elevator.currentLevel].waitingPeople) != 0 {
						longestWaiter := central.building.levels[elevator.currentLevel].waitingPeople[0]
						central.building.levels[elevator.currentLevel].waitingPeople = central.building.levels[elevator.currentLevel].waitingPeople[1:]
						elevator.people = append(elevator.people, longestWaiter)
						fmt.Println("person who wants to", longestWaiter.wantingToLevel, "went in the elevator #", elevator.identifier)
						central.building.muTex.Unlock()
					} else {
						central.building.muTex.Unlock()
						break
					}

				}

				if elevator.currentLevel < elevator.goingToLevel {
					elevator.currentLevel++
					elevator.distance++
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

// LongestWaitingorWanting : Hallo
func LongestWaitingorWanting(elevator *Elevator, central *ControlLogic) {
	longestWaitingorWantingPerson := Person{}
	if len(elevator.people) == 0 {
		for _, level := range central.building.levels {
			for _, person := range level.waitingPeople {
				if person.timeWaited > longestWaitingorWantingPerson.timeWaited {
					longestWaitingorWantingPerson = person
				}
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorWantingPerson.waitingOnLevel
	} else {
		for _, person := range elevator.people {
			if person.timeSpend >= longestWaitingorWantingPerson.timeSpend {
				longestWaitingorWantingPerson = person
			}
		}
		central.building.elevators[elevator.identifier].goingToLevel = longestWaitingorWantingPerson.wantingToLevel
	}

	if elevator.currentLevel != central.building.elevators[elevator.identifier].goingToLevel {
		fmt.Println("Elevator #", elevator.identifier, "is now on level", elevator.currentLevel, "and is going to level", central.building.elevators[elevator.identifier].goingToLevel)
	}

}

// LongestWanting : Hallo
func LongestWanting(elevator *Elevator, central *ControlLogic) {
	longestWantingPerson := Person{}
	for _, person := range elevator.people {
		if person.timeSpend > longestWantingPerson.timeSpend {
			longestWantingPerson = person
		}
	}
	elevator.goingToLevel = longestWantingPerson.wantingToLevel
	if len(elevator.people) == 0 {
		for _, level := range central.building.levels {
			for _, person := range level.waitingPeople {
				if person.timeWaited > longestWantingPerson.timeWaited {
					longestWantingPerson = person
				}
			}
		}
		elevator.goingToLevel = longestWantingPerson.waitingOnLevel
	}
	fmt.Println("Elevator #", elevator.identifier, "is now on level", elevator.currentLevel, "and is going to level", elevator.goingToLevel)
}

/* func (central *ControlLogic) toString() {
	fmt.Println()
	fmt.Println("Name:\t", central.building.name)
	fmt.Println("Capacity:\t", central.building.capacity)
	fmt.Println()
	fmt.Println("Levels:")
	for _, level := range central.building.levels {
		fmt.Println("\tIdentifier:\t", level.identifier)
		fmt.Println("\tWaiting people:\t", level.waitingPeople)
		fmt.Println()
	}
	fmt.Println("Elevators:")
	for _, elevator := range central.building.elevators {
		fmt.Println("\tIdentifier:\t", elevator.identifier)
		fmt.Println("\tCapacity:\t", elevator.capacity)
		fmt.Println("\tDistance:\t", elevator.distance)
		fmt.Println("\tPeople:\t", elevator.people)
		fmt.Println()
	}
	fmt.Println("People:")
	for _, human := range central.building.people {
		fmt.Println("\tTime waited:\t", human.timeWaited)
		fmt.Println("\tTime spend:\t", human.timeSpend)
		fmt.Println()
	}
} */
