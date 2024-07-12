// pkg/initializer/initializer.go
package initializer

import (
	"github.com/gin-gonic/gin"
	"log"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"sync"
)

var once sync.Once

// Setup initializes the configuration and logging system.
func Setup(configName string) error {
	var err error
	once.Do(func() {
		err = config.InitConfig(configName)
		if err != nil {
			log.Println("Error initializing config:", err)
			return
		}

		log.Println("config.InitConfig ok")

		if !config.Debug {
			gin.SetMode(gin.ReleaseMode)
		}

		mylog.InitLog(config.LogLevel)
		log.Println("config.LogLevel ok")
	})
	return err
}

func Cleanup() {
	mylog.Logger.Sync() // Ensure all logs are flushed properly
}
