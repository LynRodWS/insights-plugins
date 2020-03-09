package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const refreshInterval = 1 * time.Hour

var metricURL = fmt.Sprintf("%s/v0/organizations/%s/clusters/%s/data/metrics", os.Getenv("FAIRWINDS_INSIGHTS_HOST"), os.Getenv("FAIRWINDS_ORG"), os.Getenv("FAIRWINDS_CLUSTER"))
var authToken = os.Getenv("FAIRWINDS_TOKEN")

func refreshData() {
	actionItems.Reset()
	client := http.Client{}
	request, err := http.NewRequest("GET", metricURL, nil)

	if err != nil {
		panic(err)
	}
	request.Header.Add("Authorization", "Bearer "+authToken)
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	responseData := map[string]map[string]map[string]float64{}

	err = json.Unmarshal(body, &responseData)
	if err != nil {
		panic(err)
	}
	for report, reportCategories := range responseData {
		for category, categorySeverities := range reportCategories {
			for severity, value := range categorySeverities {
				actionItems.WithLabelValues(report, category, severity).Set(value)
			}
		}
	}
}

var (
	actionItems = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "fairwinds",
			Subsystem: "insights",
			Name:      "action_items",
			Help:      "A count of action items currently active for this cluster.",
		},
		[]string{
			"report",
			"category",
			"severity",
		},
	)
)

type insightsCollector struct {
}

func (c insightsCollector) Describe(ch chan<- *prometheus.Desc) {
	actionItems.Describe(ch)
}

func (c insightsCollector) Collect(ch chan<- prometheus.Metric) {
	refreshData()
	actionItems.Collect(ch)
}

func main() {
	collector := insightsCollector{}
	prometheus.DefaultRegisterer.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
