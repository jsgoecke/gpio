package main

import (
	"github.com/jsgoecke/gpio"
	"log"
	"sync"
	"time"
)

const (
	// Times is the times to iterate between on/off
	Times = 3
	// Duration is the duration, in milliseconds to pause between on/off
	Duration = 100
)

var wg sync.WaitGroup

func main() {
	ports := []int{17, 18, 23, 24, 25}
	wg.Add(len(ports))
	for _, port := range ports {
		pin, err := gpio.New(port)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("GPIO %d - Initialized\n", pin.Port)
		go blinkPort(pin)
	}
	wg.Wait()
}

func blinkPort(pin *gpio.Pin) {
	pin.Direction("out")

	for i := 0; i < Times; i++ {
		pin.High()
		event := <-pin.EventChannel
		if event.State == gpio.PinHigh {
			log.Printf("GPIO %d - State: HIGH\n", pin.Port)
		}
		time.Sleep(Duration * time.Millisecond)

		pin.Low()
		event = <-pin.EventChannel
		if event.State == gpio.PinLow {
			log.Printf("GPIO %d - State: LOW\n", pin.Port)
		}
		time.Sleep(Duration * time.Millisecond)
	}

	pin.Close()
	log.Printf("GPIO %d - Closed\n", pin.Port)
	wg.Done()
}
