package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsUseRouteTemplatesAndCountRejections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := New()
	r := gin.New()
	r.Use(m.Middleware())
	r.GET("/:shortCode", func(c *gin.Context) { c.Redirect(http.StatusFound, "https://example.com") })
	r.POST("/api/urls", func(c *gin.Context) {
		c.Set("rate_limit_rejected", true)
		c.AbortWithStatus(http.StatusTooManyRequests)
	})
	r.GET("/metrics", gin.WrapH(m.Handler()))
	for _, code := range []string{"uniqueCodeOne", "uniqueCodeTwo"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/"+code, nil))
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/urls", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/unknown/nested/path", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	body := w.Body.String()
	for _, want := range []string{"redirects_total 2", "rate_limit_rejections_total 1", `route="/:shortCode"`, `route="unmatched"`, "request_duration_seconds_bucket"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing metric %s", want)
		}
	}
	if strings.Contains(body, "uniqueCode") || strings.Contains(body, "unknown/nested") {
		t.Fatal("raw paths leaked into metric labels")
	}
}
