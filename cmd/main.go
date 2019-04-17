package main

import (
        "log"
        "flag"
        "time"
        "net/url"
        "fmt"
        "strconv"

        "github.com/eclipse/paho.mqtt.golang"
)

var publishTopic string
var lastMeasurement time.Time

// FlowRate is a constant needed to calculate rotations to liters per minute
// FlowRate = 10 corresponds to 10 l/min
var litersPerRotation float64

func createClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	//opts.SetUsername(uri.User.Username())
	//password, _ := uri.User.Password()
	//opts.SetPassword(password)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetAutoReconnect(true)
	return opts
}

func connect(clientID string, uri *url.URL) mqtt.Client {
	opts := createClientOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func listen(id string, uri *url.URL, topic string) {
	client := connect(id, uri)
	client.Subscribe(topic, 0, onMQTTMessage)
}

func onMQTTMessage(client mqtt.Client, message mqtt.Message) {
	value, err := strconv.ParseBool(string(message.Payload()))
	if err != nil {
		log.Printf("Received incorrect message payload: '%v'\n", message.Payload())
		return
	}
	if ! value {
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
	inTopic := flag.String("inTopic", "evok/input/4/value", "MQTT topic with a current pin state")
	outTopic := flag.String("outTopic", "flow/rate", "MQTT topic with a calculated flow rate")
	liters := flag.Float64("litersPerRotation", 1, "How many liters is one rotation (default: 1)")
	flag.Parse()

	publishTopic = *outTopic
	litersPerRotation = *liters
	
	brokerURL, _ := url.Parse(*broker)
	listen(*clientID, brokerURL, *inTopic)

	log.Printf("Connected to %s as %s and waiting for messages\n", *broker, *clientID)

	lastMeasurement = time.Now()
	
	// wait forever
	select{}
}
