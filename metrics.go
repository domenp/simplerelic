package simplerelic

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AppMetric is an interface for metrics reported to NewRelic
type AppMetric interface {

	// Update all the values that will be reported (or be used in calculation)
	// Called on every requests (used in gin middleware)
	Update(c *gin.Context)

	// ValueMap extracts all values from AppMetric data structures
	// to be reported to NewRelic. A single AppMetric can produce multiple
	// metrics as perceived by NewRelic.
	// Note that this function is also responsible for clearing the values
	// after they have been reported.
	ValueMap() map[string]float32
}

const (
	unknownEndpoint = "other"
)

// StandardMetric is a base for metrics dealing with endpoints
type StandardMetric struct {
	endpoints       map[string]func(urlPath string) bool
	reqCount        map[string]int
	lock            sync.RWMutex
	namePrefix      string
	allEPNamePrefix string
	metricUnit      string
}

func (m *StandardMetric) initReqCount() {
	// initialize the metrics
	for endpoint := range m.endpoints {
		m.reqCount[endpoint] = 0
	}
	m.reqCount[unknownEndpoint] = 0
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
}

// NewReqPerEndpoint creates new ReqPerEndpoint metric
func NewReqPerEndpoint(endpoints map[string]func(urlPath string) bool) *ReqPerEndpoint {

	metric := &ReqPerEndpoint{
		StandardMetric: &StandardMetric{
			endpoints:       endpoints,
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ReqPerEndpoint/",
			allEPNamePrefix: "Component/Req/overall",
			metricUnit:      "[requests]",
		},
	}

	metric.initReqCount()

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
	var allEPReq int
	for endpoint, value := range m.reqCount {
		metricName := m.namePrefix + endpoint + m.metricUnit
		metrics[metricName] = float32(value)
		allEPReq += value
		m.reqCount[endpoint] = 0
	}
	metrics[m.allEPNamePrefix+m.metricUnit] = float32(allEPReq)
	m.lock.Unlock()

	return metrics
}

/**************************************************
* Error rate per endpoint
**************************************************/

// ErrorRatePerEndpoint holds the percentage of error requests per endpoint
type ErrorRatePerEndpoint struct {
	*StandardMetric
	errorCount map[string]int
}

// NewErrorRatePerEndpoint creates new POEPerEndpoint metric
func NewErrorRatePerEndpoint(endpoints map[string]func(urlPath string) bool) *ErrorRatePerEndpoint {

	metric := &ErrorRatePerEndpoint{
		StandardMetric: &StandardMetric{
			endpoints:       endpoints,
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ErrorRatePerEndpoint/",
			allEPNamePrefix: "Component/ErrorRate/overall",
			metricUnit:      "[percent]",
		},
		errorCount: make(map[string]int),
	}

	// initialize the metrics
	metric.initReqCount()
	for endpoint := range metric.endpoints {
		metric.errorCount[endpoint] = 0
	}
	metric.errorCount[unknownEndpoint] = 0

	return metric
}

// Update the metric values
func (m *ErrorRatePerEndpoint) Update(c *gin.Context) {
	endpointName := m.endpointFromURL(c.Request.URL.Path)
	m.lock.Lock()
	if c.Writer.Status() >= 400 {
		m.errorCount[endpointName]++
	}
	m.reqCount[endpointName]++
	m.lock.Unlock()
}

// ValueMap extract all the metrics to be reported
func (m *ErrorRatePerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)

	m.lock.Lock()
	var allEPErrors int
	var allEPReq int
	for endpoint := range m.errorCount {
		metricName := m.namePrefix + endpoint + m.metricUnit

		metrics[metricName] = 0.
		if overallReq := float32(m.reqCount[endpoint]); overallReq > 0.0 {
			metrics[metricName] = float32(m.errorCount[endpoint]) / overallReq
		}

		allEPErrors += m.errorCount[endpoint]
		allEPReq += m.reqCount[endpoint]

		m.errorCount[endpoint] = 0
		m.reqCount[endpoint] = 0
	}

	metrics[m.allEPNamePrefix+m.metricUnit] = 0.
	if allEPReq > 0 {
		metrics[m.allEPNamePrefix+m.metricUnit] = float32(allEPErrors) / float32(allEPReq)
	}

	m.lock.Unlock()

	return metrics
}

/**************************************************
* Response time per endpoint
**************************************************/

// ResponseTimePerEndpoint tracks the response time per endpoint
type ResponseTimePerEndpoint struct {
	*StandardMetric
	responseTime map[string][]float32
}

// NewResponseTimePerEndpoint creates new ResponseTimePerEndpoint metric
func NewResponseTimePerEndpoint(endpoints map[string]func(urlPath string) bool) *ResponseTimePerEndpoint {

	metric := &ResponseTimePerEndpoint{
		StandardMetric: &StandardMetric{
			endpoints:       endpoints,
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ResponseTimePerEndpoint/",
			allEPNamePrefix: "Component/ResponseTime/overall",
			metricUnit:      "[ms]",
		},

		responseTime: make(map[string][]float32),
	}

	// initialize the metrics
	metric.initReqCount()
	for endpoint := range metric.endpoints {
		metric.responseTime[endpoint] = make([]float32, 1)
	}
	metric.responseTime[unknownEndpoint] = make([]float32, 1)

	return metric
}

// Update the metric values
func (m *ResponseTimePerEndpoint) Update(c *gin.Context) {

	startTime, err := c.Get("reqStartTime")
	if err != nil {
		Log.Printf("reqStart time should be time.Time")
		return
	}

	elaspsedTimeInMs := float32(time.Since(startTime.(time.Time))) / float32(time.Millisecond)

	endpointName := m.endpointFromURL(c.Request.URL.Path)
	m.lock.Lock()
	m.reqCount[endpointName]++
	m.responseTime[endpointName] = append(m.responseTime[endpointName], elaspsedTimeInMs)
	m.lock.Unlock()
}

// ValueMap extract all the metrics to be reported
func (m *ResponseTimePerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)

	m.lock.Lock()

	var allEPResponseTime float32
	var allEPReq int
	for endpoint, values := range m.responseTime {
		metricName := m.namePrefix + endpoint + m.metricUnit
		var sum float32
		for _, value := range values {
			sum += value
		}

		metrics[metricName] = 0.
		if allReq := float32(m.reqCount[endpoint]); allReq > 0 {
			metrics[metricName] = float32(sum) / allReq
		}

		allEPResponseTime += sum
		allEPReq += m.reqCount[endpoint]

		m.reqCount[endpoint] = 0
		m.responseTime[endpoint] = make([]float32, 1)
	}

	metrics[m.allEPNamePrefix+m.metricUnit] = 0.
	if allEPReq > 0 {
		metrics[m.allEPNamePrefix+m.metricUnit] = allEPResponseTime / float32(allEPReq)
	}

	m.lock.Unlock()

	return metrics
}
