package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	mqttclient "github.com/automatedhome/common/pkg/mqttclient"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var publishTopic string
var lastMeasurement time.Time

// FlowRate is a constant needed to calculate rotations to liters per minute
// FlowRate = 10 corresponds to 10 l/min
var litersPerRotation float64

func onMessage(client mqtt.Client, message mqtt.Message) {
	value, err := strconv.ParseBool(string(message.Payload()))
	if err != nil {
		log.Printf("Received incorrect message payload: '%v'\n", message.Payload())
		return
	}
	if !value {
		return
	}
	flowRate := calculate(lastMeasurement)
	client.Publish(publishTopic, 0, false, fmt.Sprintf("%f", flowRate))
}

func calculate(last time.Time) float64 {
	now := time.Now()
	duration := time.Since(lastMeasurement)

	flowRate := litersPerRotation * 60000000000 / float64(duration)

	lastMeasurement = now
	return flowRate
}

func main() {
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")
	clientID := flag.String("clientid", "flow-meter", "A clientid for the connection")
	inTopic := flag.String("inTopic", "evok/input/6/value", "MQTT topic with a current pin state")
	outTopic := flag.String("outTopic", "flow/rate", "MQTT topic to post a calculated flow rate")
	//settingsTopic := flag.String("settingsTopic", "settings/flowmeter", "MQTT topic with flowmeter settings")
	liters := flag.Float64("litersPerRotation", 0.1, "How many liters is one rotation (default: 0.1)")
	flag.Parse()

	publishTopic = *outTopic
	litersPerRotation = *liters

	brokerURL, _ := url.Parse(*broker)
	//mqttclient.New(*clientID, brokerURL, []string{*inTopic, *settingsTopic}, onMessage)
	mqttclient.New(*clientID, brokerURL, []string{*inTopic}, onMessage)

	log.Printf("Connected to %s as %s and waiting for messages\n", *broker, *clientID)

	lastMeasurement = time.Now()

	// wait forever
	select {}
}
