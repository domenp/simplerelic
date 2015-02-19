package simplerelic

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// SimpleReporter reports metrics to NewRelic
	SimpleReporter *Reporter

	// DefaultEndpoints contains default endpoints defined for standard metrics
	DefaultEndpoints map[string]func(urlPath string) bool

	// NewRelic GUID for creating the NewRelic plugin
	Guid string
)

func init() {
	DefaultEndpoints = make(map[string]func(urlPath string) bool)
}

func onReqStartHandler(c *gin.Context) {
	c.Set("reqStartTime", time.Now())
}

func onReqEndHandler(c *gin.Context) {
	for _, v := range SimpleReporter.metrics {
		v.Update(c)
	}
}

// Handler is a gin middleware that updates metrics
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		onReqStartHandler(c)
		c.Next()
		onReqEndHandler(c)
	}
}

// InitDefaultReporter creates a new reporter and adds standard metrics
func InitDefaultReporter(appname string, licence string, verbose bool) (*Reporter, error) {

	var err error
	SimpleReporter, err = NewReporter(appname, licence, verbose)
	if err != nil {
		return nil, err
	}

	if len(DefaultEndpoints) < 1 {
		return nil, errors.New("No endpoints defined, at least one needed")
	}

	SimpleReporter.AddMetric(NewReqPerEndpoint(DefaultEndpoints))
	SimpleReporter.AddMetric(NewErrorRatePerEndpoint(DefaultEndpoints))
	SimpleReporter.AddMetric(NewResponseTimePerEndpoint(DefaultEndpoints))

	return SimpleReporter, nil
}

// AddDefaultEndpoint adds an endpoint consisting of name and a matcher function
// associating the given url with the endpoint
func AddDefaultEndpoint(name string, matcherFunc func(urlPath string) bool) {
	DefaultEndpoints[name] = matcherFunc
}
