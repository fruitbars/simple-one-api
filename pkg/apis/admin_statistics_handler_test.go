package apis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminStatisticsQueryRejectsInvalidParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name    string
		query   string
		message string
	}{
		{name: "bucket", query: "bucket=minute", message: "bucket must be hour or day"},
		{name: "range", query: "from=2026-08-23T12%3A00%3A00Z&to=2026-08-23T11%3A00%3A00Z", message: "from must be earlier than to"},
	} {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/statistics/overview?"+test.query, nil)
			_, err := adminStatisticsQuery(context)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}
