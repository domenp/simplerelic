package simplerelic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	endpointName  = "log"
	componentName = "Component/ResponseTimePerEndpoint/"
)

func TestResponseTimeValueMap(t *testing.T) {

	req, _ := http.NewRequest("GET", "/log", nil)
	recorder := httptest.NewRecorder()

	r := gin.New()

	AddDefaultEndpoint(
		endpointName,
		func(urlPath string) bool { return strings.HasPrefix(urlPath, "/log") },
	)
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
	if len(values) != 3 {
		t.Errorf("error: not enough metrics received")
	}

	if _, ok := values[componentName+endpointName+"[ms]"]; !ok {
		t.Errorf("error: other endpoint not found")
	}

	if _, ok := values["Component/ResponseTime/overall[ms]"]; !ok {
		t.Errorf("error: overall metric not found")
	}

	// check the response time calculation
	for name, value := range values {
		if strings.HasSuffix(name, endpointName+"[ms]") {
			if value != 0.15 {
				t.Errorf("error: expected %f, got %f", 0.15, value)
			}
		}
		if strings.HasSuffix(name, "overall[ms]") {
			if value != 0.15 {
				t.Errorf("error: expected %f, got %f", 0.15, value)
			}
		}
	}

	// check if the metrics are cleared
	for _, value := range m.ValueMap() {
		if value != 0. {
			t.Errorf("error: expected %f, got %f", 0., value)
		}
	}
}
