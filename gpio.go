package gpio

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

const (
	// Export is the file to export to get control of GPIO
	Export = "/sys/class/gpio/export"
	// Unexport is the file to uneexport to release control of GPIO
	Unexport = "/sys/class/gpio/unexport"
	// Path is the prefix for the gpio port to export
	Path = "/sys/class/gpio/gpio"
	// PinHigh is the pin being in the high state
	PinHigh = 48
	// PinLow is the in being in the low state
	PinLow = 49
	// Done is code used to signal a channel to complete the go routine
	Done = 104
)

// Pin represents a GPIO port
type Pin struct {
	Port           int
	Fd             int
	Export         string
	Unexport       string
	Path           string
	ValuePath      string
	DirectionPath  string
	State          byte
	CommandChannel chan byte
	EventChannel   chan *Event
}

// Event is an event from the file system
type Event struct {
	State byte
}

// New returns a new pin
func New(port int) (*Pin, error) {
	pin := &Pin{
		Port:           port,
		Export:         Export,
		Unexport:       Unexport,
		Path:           Path + strconv.Itoa(port),
		ValuePath:      Path + strconv.Itoa(port) + "/value",
		DirectionPath:  Path + strconv.Itoa(port) + "/direction",
		State:          49,
		CommandChannel: make(chan byte),
		EventChannel:   make(chan *Event),
	}

	fd, err := openFh(pin.Export)
	if err != nil {
		return nil, err
	}

	fmt.Fprint(fd, pin.Port)
	fd.Close()

	_, err = os.Stat(pin.Path)
	if err != nil {
		return nil, err
	}

	pin.eventProcessor()
	go pin.commandProcessor()
	return pin, nil
}

// eventProcessor processes events from the file system
func (pin *Pin) eventProcessor() error {
	fd, err := syscall.InotifyInit()
	pin.Fd = fd
	if err != nil {
		return err
	}

	_, err = syscall.InotifyAddWatch(fd, pin.ValuePath, syscall.IN_MODIFY)
	if err != nil {
		return err
	}

	go pin.watcher()
	return nil
}

// watcher tracks the changes to the value file of a pin
func (pin *Pin) watcher() {
	buf := make([]byte, syscall.SizeofInotifyEvent)
	n, err := syscall.Read(pin.Fd, buf)
	for err == nil && n == syscall.SizeofInotifyEvent {
		data, _ := ioutil.ReadFile(pin.ValuePath)
		pin.State = data[0]
		pin.EventChannel <- &Event{State: pin.State}
		n, err = syscall.Read(pin.Fd, buf)
	}
}

// commandProcessor is a go routine to handle the High/Low commands for a pin
func (pin *Pin) commandProcessor() {
	for {
		value := <-pin.CommandChannel
		if value == Done {
			fh, _ := openFh(pin.ValuePath)
			fmt.Fprint(fh, 0)
			break
		}
		fh, _ := openFh(pin.ValuePath)
		fmt.Fprint(fh, string(value))
		fh.Close()
	}
}

// Direction of the pin to use
func (pin *Pin) Direction(direction string) {
	fd, _ := openFh(pin.DirectionPath)
	fmt.Fprint(fd, direction)
	fd.Close()
}

// Close a pin
func (pin *Pin) Close() {
	syscall.InotifyRmWatch(pin.Fd, syscall.IN_MODIFY)
	pin.CommandChannel <- Done
	fd, _ := openFh(pin.Unexport)
	fd.Close()
}

// High turns the pin to the high state
func (pin *Pin) High() {
	pin.CommandChannel <- PinHigh
}

// Low turns the pin to the low state
func (pin *Pin) Low() {
	pin.CommandChannel <- PinLow
}

// openFh opens a file and returns the file handle
func openFh(filename string) (*os.File, error) {
	fh, err := os.OpenFile(filename, os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return nil, err
	}
	return fh, nil
}
