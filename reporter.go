package simplerelic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	newrelicURL = "https://platform-api.newrelic.com/platform/v1/metrics"
)

// Reporter keeps track of the app metrics and sends them to NewRelic
type Reporter struct {
	metrics  []AppMetric
	host     string
	pid      int
	guid     string
	duration int
	version  string
	appName  string
	licence  string
	verbose  bool
}

type newRelicData struct {
	Agent      *newRelicAgent       `json:"agent"`
	Components []*newRelicComponent `json:"components"`
}

type newRelicAgent struct {
	Host    string `json:"host"`
	Pid     int    `json:"pid"`
	Version string `json:"version"`
}

type newRelicComponent struct {
	Name     string             `json:"name"`
	Guid     string             `json:"guid"`
	Duration int                `json:"duration"`
	Metrics  map[string]float32 `json:"metrics"`
}

// NewReporter creates a new Reporter
func NewReporter(appName string, licence string, verbose bool) (*Reporter, error) {

	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("Can not get hostname")
	}

	pid := os.Getpid()

	if licence == "" {
		return nil, errors.New("Please specify Newrelic licence")
	}

	reporter := &Reporter{
		host:     host,
		pid:      pid,
		guid:     Guid,
		duration: 60,
		appName:  appName,
		licence:  licence,
		version:  "1.0.0",
		verbose:  verbose,
		metrics:  make([]AppMetric, 0, 5),
	}

	return reporter, nil
}

// Start sending metrics to NewRelic
func (reporter *Reporter) Start() {

	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("SimpleRelic reporter crashed")
		}
	}()

	ticker := time.NewTicker(time.Second * 60)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				reporter.sendMetrics()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// AddMetric adds a new metric to be reported
func (reporter *Reporter) AddMetric(metric AppMetric) {
	reporter.metrics = append(reporter.metrics, metric)
}

// extract and send metrics to NewRelic
func (reporter *Reporter) sendMetrics() {

	reqData := reporter.prepareReqData()

	// extract all metrics to be sent to NewRelic
	// from the AppMetric data structure
	for _, metrics := range reporter.metrics {
		for k, m := range metrics.ValueMap() {
			reqData.Components[0].Metrics[k] = m
		}
		metrics.Clear()
	}

	json, err := json.Marshal(reqData)
	if err != nil {
		fmt.Errorf("error marshaling json")
	}

	if reporter.verbose {
		fmt.Println("sending metrics to NewRelic")
		fmt.Println(string(json))
	}

	reporter.doRequest(json)
}

func (reporter *Reporter) prepareReqData() *newRelicData {
	reqData := &newRelicData{
		Agent: &newRelicAgent{
			Host:    reporter.host,
			Pid:     reporter.pid,
			Version: reporter.version,
		},
		Components: []*newRelicComponent{
			&newRelicComponent{
				Name:     reporter.appName,
				Guid:     reporter.guid,
				Duration: reporter.duration,
				Metrics:  make(map[string]float32),
			},
		},
	}

	reqData.Components[0] = &newRelicComponent{
		Name:     reporter.appName,
		Guid:     reporter.guid,
		Duration: reporter.duration,
		Metrics:  make(map[string]float32),
	}

	return reqData
}

func (reporter *Reporter) doRequest(json []byte) {
	req, err := http.NewRequest("POST", newrelicURL, bytes.NewReader(json))
	if err != nil {
		fmt.Errorf("error setting up newrelic request")
	}
	req.Header.Set("X-License-Key", reporter.licence)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Errorf("Post request to NewRelic failed")
		return
	}
	defer resp.Body.Close()

	if reporter.verbose {
		responseJson, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Errorf("reading of NewRelic response failed")
		}
		fmt.Println("response from NewRelic")
		fmt.Println(string(responseJson))
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("Error in request to NewRelic, status code %d", resp.StatusCode)
	}
}
