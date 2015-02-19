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

		m.reqCount[endpointName]++
		for i := 0; i < 4; i++ {
			m.responseTime[endpointName] = append(m.responseTime[endpointName], 0.2)
		}
	})

	for name, value := range m.ValueMap() {
		if strings.HasSuffix(name, endpointName+"[ms]") {
			if value != 0.2 {
				t.Errorf("error: expected %f, got %f", 0.2, value)
			}
		}
	}

	r.ServeHTTP(recorder, req)
}
