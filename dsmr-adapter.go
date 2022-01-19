package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Device info
// http://[ip]/api/v1/dev/info

// Meter info
// http://[ip]/api/v1/sm/info

// Actual
// http://[ip]/api/v1/sm/actual

var (
	reg       = prometheus.NewRegistry()
	namespace = "dsmr"

	voltage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "voltage",
		Help:      "Current voltage in V",
	}, []string{"phase"})

	power = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "power_delivered",
		Help:      "Current power delivery in kW",
	}, []string{"direction", "phase"})

	energyTransported = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "energy_transported",
		Help:      "Energy total in kWh",
	}, []string{"direction", "tariff"})

	gasDelivered = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gas_delivered",
		Help:      "Gas delivered in m3",
	})

	baseUrl = os.Getenv("DSMR_BASE_URL")
)

func main() {
	if baseUrl == "" {
		fmt.Fprintf(os.Stderr, "DSMR_BASE_URL is empty")
		os.Exit(1)
	}

	reg.MustRegister(voltage)
	reg.MustRegister(power)
	reg.MustRegister(energyTransported)
	reg.MustRegister(gasDelivered)

	http.HandleFunc("/metrics", handle)

	log.Printf("Server is listening on port 8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handle(w http.ResponseWriter, req *http.Request) {
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/sm/actual", baseUrl))

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	var response Response

	json.Unmarshal([]byte(body), &response)

	for _, element := range response.Actual {
		// Debug log the values from the API
		// log.Printf("%s %v %s", element.Name, element.Value, element.Unit)

		if strings.HasPrefix(element.Name, "voltage_") {
			voltage.With(prometheus.Labels{"phase": strings.Replace(element.Name, "voltage_", "", 1)}).Set(element.Value)
		}

		if strings.HasPrefix(element.Name, "power_delivered_") {
			power.With(prometheus.Labels{
				"phase":     strings.Replace(element.Name, "power_delivered_", "", 1),
				"direction": "delivered",
			}).Set(element.Value)
		}

		if strings.HasPrefix(element.Name, "power_returned_") {
			power.With(prometheus.Labels{
				"phase":     strings.Replace(element.Name, "power_returned_", "", 1),
				"direction": "returned",
			}).Set(element.Value)
		}

		if strings.HasPrefix(element.Name, "energy_") {
			direction := "returned"

			if strings.Contains(element.Name, "_delivered_") {
				direction = "delivered"
			}

			tariff := "low"

			if strings.HasSuffix(element.Name, "2") {
				tariff = "high"
			}

			energyTransported.With(prometheus.Labels{
				"direction": direction,
				"tariff":    tariff,
			}).Set(element.Value)
		}

		if element.Name == "gas_delivered" {
			gasDelivered.Set(element.Value)
		}
	}

	handler.ServeHTTP(w, req)
}

type Response struct {
	Actual []Measurement
}

type Measurement struct {
	Name  string
	Value float64
	Unit  string
}
