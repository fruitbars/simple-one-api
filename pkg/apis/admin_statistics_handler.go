package apis

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/initializer"
	"simple-one-api/pkg/statistics"
)

func AdminStatisticsOverviewHandler(c *gin.Context) {
	service := initializer.StatisticsService()
	if service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "statistics service is unavailable"})
		return
	}
	query, err := adminStatisticsQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	overview, err := service.Overview(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query statistics: " + err.Error()})
		return
	}
	labelStatisticsAccessKeys(&overview)
	c.JSON(http.StatusOK, overview)
}

func labelStatisticsAccessKeys(overview *statistics.Overview) {
	labels := map[string]string{"anonymous": "未认证请求"}
	configuration := config.CurrentConfiguration()
	if configuration.APIKey != "" {
		labels[statistics.Fingerprint(configuration.APIKey)] = "主密钥"
	}
	for index, item := range configuration.APIKeys {
		if item.APIKey != "" {
			labels[statistics.Fingerprint(item.APIKey)] = fmt.Sprintf("访问密钥 %d", index+1)
		}
	}
	for index := range overview.AccessKeys {
		if label, ok := labels[overview.AccessKeys[index].ID]; ok {
			overview.AccessKeys[index].Name = label
		}
	}
}

func AdminStatisticsExportHandler(c *gin.Context) {
	service := initializer.StatisticsService()
	if service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "statistics service is unavailable"})
		return
	}
	query, err := adminStatisticsQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="simple-one-api-statistics.csv"`)
	if err := service.ExportCSV(c.Request.Context(), query, c.Writer); err != nil {
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "export statistics: " + err.Error()})
		}
	}
}

func adminStatisticsQuery(c *gin.Context) (statistics.Query, error) {
	query := statistics.Query{
		Bucket:    strings.ToLower(strings.TrimSpace(c.Query("bucket"))),
		Provider:  strings.TrimSpace(c.Query("provider")),
		Model:     strings.TrimSpace(c.Query("model")),
		Protocol:  strings.TrimSpace(c.Query("protocol")),
		AccessKey: strings.TrimSpace(c.Query("access_key")),
		Status:    strings.ToLower(strings.TrimSpace(c.Query("status"))),
	}
	var err error
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		query.From, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return statistics.Query{}, fmt.Errorf("from must be an RFC3339 timestamp")
		}
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		query.To, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return statistics.Query{}, fmt.Errorf("to must be an RFC3339 timestamp")
		}
	}
	if query.Status != "" && query.Status != "success" && query.Status != "failure" {
		return statistics.Query{}, fmt.Errorf("status must be success or failure")
	}
	if query.Bucket != "" && query.Bucket != "hour" && query.Bucket != "day" {
		return statistics.Query{}, fmt.Errorf("bucket must be hour or day")
	}
	if !query.From.IsZero() && !query.To.IsZero() && !query.From.Before(query.To) {
		return statistics.Query{}, fmt.Errorf("from must be earlier than to")
	}
	return query, nil
}
