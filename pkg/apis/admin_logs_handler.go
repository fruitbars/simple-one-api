package apis

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/mylog"
)

func AdminLogsHandler(c *gin.Context) {
	after, _ := strconv.ParseUint(c.Query("after"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	c.JSON(http.StatusOK, gin.H{"data": mylog.LiveEntries(after, limit)})
}
