package main

import (
	"log"

	"github.com/muka/bluez-client/api"
	"github.com/muka/bluez-client/devices"
	"github.com/op/go-logging"
	"github.com/tj/go-debug"
)

var logger = logging.MustGetLogger("main")
var dbg = debug.Debug("bluez:main")

var adapterID = "hci0"
var tagAddress = "B0:B4:48:C9:4B:01"

func main() {

	dev, err := api.GetDeviceByAddress(tagAddress)
	if err != nil {
		panic(err)
	}

	if dev == nil {
		panic("Device not found")
	}

	err = dev.Connect()
	if err != nil {
		panic(err)
	}

	sensorTag, err := devices.NewSensorTag(dev)
	if err != nil {
		panic(err)
	}

	temp, err := sensorTag.Temperature.Read()
	if err != nil {
		panic(err)
	}

	log.Printf("Temperature %v°", temp)
}
