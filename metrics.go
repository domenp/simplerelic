package simplerelic

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// AppMetric is an interface for metrics to be reported
type AppMetric interface {

	// Update the values on every requests (used in gin middleware)
	Update(c *gin.Context)

	// Clear the values (after they are reported)
	Clear()

	// ValueMap extracts all values to be reported to NewRelic
	ValueMap() map[string]float32
}

/************************************
 * requests per endpoint
 ***********************************/

// ReqPerEndpoint holds number of requests per endpoint
type ReqPerEndpoint struct {
	reqCount     map[string]int
	reqCountLock sync.RWMutex
	endpoints    map[string]func(urlPath string) bool
	namePrefix   string
	metricUnit   string
}

// NewReqPerEndpoint creates new ReqPerEndpoint metric
func NewReqPerEndpoint(endpoints map[string]func(urlPath string) bool) *ReqPerEndpoint {

	metric := &ReqPerEndpoint{
		reqCount:   make(map[string]int),
		endpoints:  endpoints,
		namePrefix: "Component/ReqPerEndpoint/",
		metricUnit: "[requests]",
	}
	// initialize the metrics
	metric.Clear()

	return metric
}

// Update the metric values
func (self *ReqPerEndpoint) Update(c *gin.Context) {
	urlPath := c.Request.URL.Path
	endpointName := self.endpointFromURL(urlPath)
	self.reqCountLock.Lock()
	self.reqCount[endpointName]++
	self.reqCountLock.Unlock()
}

// Clear the metric values
func (self *ReqPerEndpoint) Clear() {
	self.reqCountLock.Lock()
	for endpoint := range self.endpoints {
		self.reqCount[endpoint] = 0
	}
	self.reqCount["other"] = 0
	self.reqCountLock.Unlock()
}

// ValueMap extract all the metrics to be reported
func (self *ReqPerEndpoint) ValueMap() map[string]float32 {
	metrics := make(map[string]float32)
	for endpoint, count := range self.reqCount {
		metricName := self.namePrefix + endpoint + self.metricUnit
		metrics[metricName] = float32(count)
	}

	return metrics
}

// endpointFromUrl returns name of the endpoint that matches first
// if none of them matches it returns "other"
func (self *ReqPerEndpoint) endpointFromURL(urlPath string) string {
	for name, isMatchFunc := range self.endpoints {
		if isMatchFunc(urlPath) {
			return name
		}
	}

	return "other"
}
