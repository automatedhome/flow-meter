package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type EvokDevice struct {
	Value   int    `json:"value"`
	Circuit string `json:"circuit"`
	Dev     string `json:"dev"`
}

var (
	lastMeasurement   time.Time
	litersPerRotation float64
	lastPass          time.Time
	evokCircuit       string
	evokAddress       string
)

var (
	wsConnectionCloses = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "solar_flow",
		Name:      "websocket_connection_closes_total",
		Help:      "Total number of websocket connection closes",
	})
	flow = promauto.NewGauge(prometheus.GaugeOpts{
		// TODO(paulfantom): change this to m^3 per second to conform to SI
		Name: "rate_liters_per_minute",
		Help: "Current flow rate in liters per minute",
	})
	liters = promauto.NewCounter(prometheus.CounterOpts{
		// TODO(paulfantom): change this to m^3 to conform to SI
		Name: "liters_total",
		Help: "Current number of liters circulated",
	})
)

func calculate() float64 {
	now := time.Now()
	duration := time.Since(lastMeasurement)

	// 60000000000 is needed to convert value to l/min
	flowRate := litersPerRotation / float64(duration) * 60000000000

	lastMeasurement = now
	return flowRate
}

func digitalInput(address string, circuit string) {
	fmt.Printf("Connecting to EVOK at %s and handling updates for circuit %s\n", address, circuit)

	conn, _, _, err := ws.DefaultDialer.Dial(context.TODO(), evokAddress)
	if err != nil {
		panic("Connecting to EVOK failed: " + err.Error())
	}
	defer conn.Close()

	msg := "{\"cmd\":\"filter\", \"devices\":[\"input\"]}"
	if err = wsutil.WriteClientMessage(conn, ws.OpText, []byte(msg)); err != nil {
		panic("Sending websocket message to EVOK failed: " + err.Error())
	}

	var inputs []EvokDevice
	for {
		payload, err := wsutil.ReadServerText(conn)
		if err != nil {
			if err == io.EOF {
				log.Println("Websocket connection closed.")
				wsConnectionCloses.Inc()
				break
			}
			log.Printf("Received incorrect data: %#v", err)
			continue
		}

		if err := json.Unmarshal(payload, &inputs); err != nil {
			log.Printf("Could not parse received data: %#v", err)
			continue
		}

		if inputs[0].Circuit == evokCircuit && inputs[0].Value == 1 {
			flow.Set(calculate())
			liters.Add(litersPerRotation)
		}
	}
}

func httpHealthCheck(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(1 * time.Minute)
	if lastPass.Add(timeout).After(time.Now()) {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
}

func init() {
	liters := flag.Float64("liters-per-rotation", 0.1, "How many liters is one rotation (default: 0.1 l/rot)")
	addr := flag.String("evok-address", "ws://localhost:8080/ws", "EVOK websocket API address (default: ws://localhost:8080/ws)")
	circuit := flag.Int("evok-circuit", 1, "EVOK digital input circuit to which sensor is connected (default: 1)")
	flag.Parse()

	evokCircuit = strconv.Itoa(*circuit)
	evokAddress = *addr

	litersPerRotation = *liters
	lastMeasurement = time.Now()
}

func main() {
	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())
	// Expose healthcheck
	http.HandleFunc("/health", httpHealthCheck)
	go func() {
		if err := http.ListenAndServe(":7000", nil); err != nil {
			panic("HTTP Server failed: " + err.Error())
		}
	}()

	go digitalInput(evokAddress, evokCircuit)

	for {
		time.Sleep(15 * time.Second)
		lastPass = time.Now()
	}
}
