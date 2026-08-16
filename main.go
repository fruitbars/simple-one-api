package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"simple-one-api/pkg/appserver"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/initializer"
	"simple-one-api/pkg/mylog"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	configName := "config.json"
	if len(os.Args) > 1 {
		configName = os.Args[1]
	}

	if err := initializer.Setup(configName); err != nil {
		return
	}
	defer initializer.Cleanup()

	router := appserver.NewRouter()
	server := &http.Server{
		Addr:              config.CurrentServerPort(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		mylog.Logger.Error(err.Error())
	}
}
