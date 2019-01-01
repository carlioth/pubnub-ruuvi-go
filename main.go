package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	pubnub "github.com/pubnub/go"
)

var (
	httpURL      = os.Getenv("HTTP_URL")
	outdoor      = "DA:FA:48:39:79:97"
	outdoorSent  = time.Now().Unix()
	indoor       = "FD:D4:BA:BE:26:3A"
	indoorSent   = time.Now().Unix()
	basement     = "D5:04:AA:85:48:79"
	basementSent = time.Now().Unix()
)

var pn *pubnub.PubNub

func init() {
	config := pubnub.NewConfig()
	config.SubscribeKey = "-SUB-KEY-"
	config.PublishKey = "-PUB-KEY-"
	pn = pubnub.NewPubNub(config)
}

type SensorData struct {
	Temp      float64
	Humidity  float64
	Pressure  uint32
	Battery   uint16
	Address   string
	TimeStamp int64
	Name      string
}

type SensorFormat3 struct {
	ManufacturerID      uint16
	DataFormat          uint8
	Humidity            uint8
	Temperature         uint8
	TemperatureFraction uint8
	Pressure            uint16
	BatteryVoltageMv    uint16
}

func setSensorName(data *SensorData) *SensorData {
	switch data.Address {
	case indoor:
		data.Name = "indoor"
	case outdoor:
		data.Name = "outdoor"
	case basement:
		data.Name = "basement"
	default:
		data.Name = "Unknown"
	}
	return data
}

func sendSensorData(data *SensorData) {
	switch data.Name {
	case "indoor":
		if (indoorSent + 60) < time.Now().Unix() {
			s, _ := json.Marshal(data)
			pn.Publish().
				Channel(data.Name).
				Message(string(s)).
				Execute()
			indoorSent = time.Now().Unix()
			fmt.Printf("Sent message from %s \n", data.Name)
		}
	case "outdoor":
		if (outdoorSent + 60) < time.Now().Unix() {
			s, _ := json.Marshal(data)
			pn.Publish().
				Channel(data.Name).
				Message(string(s)).
				Execute()
			outdoorSent = time.Now().Unix()
			fmt.Printf("Sent message from %s \n", data.Name)
		}
	case "basement":
		if (basementSent + 60) < time.Now().Unix() {
			s, _ := json.Marshal(data)
			pn.Publish().
				Channel(data.Name).
				Message(string(s)).
				Execute()
			basementSent = time.Now().Unix()
			fmt.Printf("Sent message from %s \n", data.Name)
		}
	}
}

func parseTemperature(t uint8, f uint8) float64 {
	var mask uint8
	mask = (1 << 7)
	isNegative := (t & mask) > 0
	temp := float64(t&^mask) + float64(f)/100.0
	if isNegative {
		temp *= -1
	}
	return temp
}

func parseSensorFormat3(data []byte) *SensorData {
	reader := bytes.NewReader(data)
	result := SensorFormat3{}
	err := binary.Read(reader, binary.BigEndian, &result)
	if err != nil {
		panic(err)
	}

	sensorData := SensorData{}
	sensorData.Temp = parseTemperature(result.Temperature, result.TemperatureFraction)
	sensorData.Humidity = float64(result.Humidity) / 2.0
	sensorData.Pressure = uint32(result.Pressure) + 50000
	sensorData.Battery = result.BatteryVoltageMv
	return &sensorData
}

//ParseRuuviData parses ruuvidata
func ParseRuuviData(data []byte, a string) {

	sendData := func(sensorData *SensorData) {
		sensorData.Address = a
		sensorData = setSensorName(sensorData)
		sensorData.TimeStamp = time.Now().Unix()
		sendSensorData(sensorData)
	}

	if len(data) == 16 && binary.LittleEndian.Uint16(data[0:2]) == 0x0499 {
		sensorFormat := data[2]
		switch sensorFormat {
		case 3:
			sendData(parseSensorFormat3(data))
		default:
			fmt.Printf("Unknown sensor format %d", sensorFormat)
		}
	} else {
		// fmt.Printf("Not a ruuvi device \n")
	}

}

func onStateChanged(device gatt.Device, s gatt.State) {
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("Scanning for iBeacon Broadcasts...")
		device.Scan([]gatt.UUID{}, true)
		return
	default:
		device.StopScanning()
	}
}

func onPeripheralDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	ParseRuuviData(a.ManufacturerData, p.ID())
}

func main() {
	device, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	device.Handle(gatt.PeripheralDiscovered(onPeripheralDiscovered))
	device.Init(onStateChanged)
	select {}
}
