package simplerelic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	endpointName = "log"
)

var (
	req      *http.Request
	recorder *httptest.ResponseRecorder
	r        *gin.Engine
)

func setup() {
	req, _ = http.NewRequest("GET", "/log", nil)
	recorder = httptest.NewRecorder()

	r = gin.New()

	AddDefaultEndpoint(
		endpointName,
		func(urlPath string) bool { return strings.HasPrefix(urlPath, "/log") },
	)
}

func testPresence(t *testing.T, metricName string, allEPName string, metricUnit string, values map[string]float32) {

	if len(values) != 3 {
		t.Errorf("error: not enough metrics received")
	}

	if _, ok := values[metricName+"other"+metricUnit]; !ok {
		t.Errorf("error: other endpoint not found")
	}

	if _, ok := values[allEPName+metricUnit]; !ok {
		t.Errorf("error: overall metric not found")
	}
}

func checkCalc(t *testing.T, values map[string]float32, expected float32) {
	for name, value := range values {
		if strings.HasSuffix(name, endpointName+"[percent]") {
			if value != expected {
				t.Errorf("error: expected %f, got %f", expected, value)
			}
		}
		if strings.HasSuffix(name, "overall[percent]") {
			if value != expected {
				t.Errorf("error: expected %f, got %f", expected, value)
			}
		}
	}
}

func checkIsCleared(t *testing.T, m AppMetric) {
	// check if the metrics are cleared
	for _, value := range m.ValueMap() {
		if value != 0. {
			t.Errorf("error: expected %f, got %f", 0., value)
		}
	}
}

func TestReq(t *testing.T) {

	setup()

	m := NewReqPerEndpoint(DefaultEndpoints)

	r.GET("/log", func(c *gin.Context) {
		m.Update(c)
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()
	testPresence(
		t,
		"Component/ReqPerEndpoint/",
		"Component/Req/overall",
		"[requests]",
		values,
	)

	// check the error rate calculation
	checkCalc(t, values, 1)
	checkIsCleared(t, m)

}

func TestErrorRate(t *testing.T) {

	setup()

	m := NewErrorRatePerEndpoint(DefaultEndpoints)

	r.GET("/log", func(c *gin.Context) {
		for i := 0; i < 4; i++ {
			c.Writer.WriteHeader(404)
			m.Update(c)
		}
		for i := 0; i < 4; i++ {
			c.Writer.WriteHeader(200)
			m.Update(c)
		}
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()
	testPresence(
		t,
		"Component/ErrorRatePerEndpoint/",
		"Component/ErrorRate/overall",
		"[percent]",
		values,
	)

	// check the error rate calculation
	checkCalc(t, values, 0.5)
	checkIsCleared(t, m)
}

func TestResponseTimeValueMap(t *testing.T) {

	setup()

	m := NewResponseTimePerEndpoint(DefaultEndpoints)

	r.GET("/log", func(c *gin.Context) {

		ts := []float32{0.1, 0.2, 0.1, 0.2}
		for _, t := range ts {
			m.responseTime[endpointName] = append(m.responseTime[endpointName], t)
			m.reqCount[endpointName]++
		}
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()
	testPresence(
		t,
		"Component/ResponseTimePerEndpoint/",
		"Component/ResponseTime/overall",
		"[ms]",
		values,
	)

	// check the response time calculation
	checkCalc(t, values, 0.15)
	checkIsCleared(t, m)
}
