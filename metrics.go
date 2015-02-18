package simplerelic

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// AppMetric is an interface for metrics reported to NewRelic
type AppMetric interface {

	// Update the values on every requests (used in gin middleware)
	Update(c *gin.Context)

	// ValueMap extracts all values to be reported to NewRelic
	// Note that this function is also responsible for clearing the values
	// after they have been reported.
	ValueMap() map[string]float32
}

const (
	unknownEndpoint = "other"
)

// StandardMetric is a base for metrics dealing with endpoints
type StandardMetric struct {
	endpoints map[string]func(urlPath string) bool
}

// endpointFromUrl returns name of the endpoint that matches first
// if none of them matches it returns "other"
func (m *StandardMetric) endpointFromURL(urlPath string) string {
	for name, isMatchFunc := range m.endpoints {
		if isMatchFunc(urlPath) {
			return name
		}
	}

	return unknownEndpoint
}

/************************************
 * requests per endpoint
 ***********************************/

// ReqPerEndpoint holds number of requests per endpoint
type ReqPerEndpoint struct {
	*StandardMetric
	reqCount   map[string]int
	lock       sync.RWMutex
	namePrefix string
	metricUnit string
}

// NewReqPerEndpoint creates new ReqPerEndpoint metric
func NewReqPerEndpoint(endpoints map[string]func(urlPath string) bool) *ReqPerEndpoint {

	metric := &ReqPerEndpoint{
		StandardMetric: &StandardMetric{endpoints: endpoints},
		reqCount:       make(map[string]int),
		namePrefix:     "Component/ReqPerEndpoint/",
		metricUnit:     "[requests]",
	}

	// initialize the metrics
	for endpoint := range metric.endpoints {
		metric.reqCount[endpoint] = 0
	}
	metric.reqCount[unknownEndpoint] = 0

	return metric
}

// Update the metric values
func (m *ReqPerEndpoint) Update(c *gin.Context) {
	endpointName := m.endpointFromURL(c.Request.URL.Path)
	m.lock.Lock()
	m.reqCount[endpointName]++
	m.lock.Unlock()
}

// ValueMap extract all the metrics to be reported
func (m *ReqPerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)
	m.lock.Lock()
	for endpoint, value := range m.reqCount {
		metricName := m.namePrefix + endpoint + m.metricUnit
		metrics[metricName] = float32(value)
		m.reqCount[endpoint] = 0
	}
	m.lock.Unlock()

	return metrics
}

/**************************************************
* Percentage of errors per endpoint
**************************************************/

// POEPerEndpoint holds the percentage of error requests per endpoint
type POEPerEndpoint struct {
	*StandardMetric
	reqCount   map[string]int
	errorCount map[string]int
	lock       sync.RWMutex
	namePrefix string
	metricUnit string
}

// NewPOEPerEndpoint creates new POEPerEndpoint metric
func NewPOEPerEndpoint(endpoints map[string]func(urlPath string) bool) *POEPerEndpoint {

	metric := &POEPerEndpoint{
		StandardMetric: &StandardMetric{endpoints: endpoints},
		errorCount:     make(map[string]int),
		reqCount:       make(map[string]int),
		namePrefix:     "Component/PercentageOfErrorsPerEndpoint/",
		metricUnit:     "[percent]",
	}
	// initialize the metrics
	for endpoint := range metric.endpoints {
		metric.errorCount[endpoint] = 0
		metric.reqCount[endpoint] = 0
	}
	metric.errorCount[unknownEndpoint] = 0
	metric.reqCount[unknownEndpoint] = 0

	return metric
}

// Update the metric values
func (m *POEPerEndpoint) Update(c *gin.Context) {
	endpointName := m.endpointFromURL(c.Request.URL.Path)
	m.lock.Lock()
	if c.Writer.Status() >= 400 {
		m.errorCount[endpointName]++
	}
	m.reqCount[endpointName]++
	m.lock.Unlock()
}

// ValueMap extract all the metrics to be reported
func (m *POEPerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)

	m.lock.Lock()
	for endpoint := range m.errorCount {
		metricName := m.namePrefix + endpoint + m.metricUnit
		if overallReq := float32(m.reqCount[endpoint]); overallReq > 0.0 {
			metrics[metricName] = float32(m.errorCount[endpoint]) / overallReq
		}
		m.errorCount[endpoint] = 0
		m.reqCount[endpoint] = 0
	}
	m.lock.Unlock()

	return metrics
}
