/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * This is main file of the oai-nwdaf-nbi-analytics HTTP server.
 */

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openairinterface/oai-cn5g-nwdaf/components/oai-nwdaf-nbi-analytics/internal/analytics"
)

type MainConfig struct {
	Server struct {
		Addr string `envconfig:"SERVER_ADDR"`
	}
}

func main() {
	// load the environment variables from the file .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	var config MainConfig
	err = envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	// Initialize internal package
	analytics.InitConfig()
	// Create router
	NWDAFAnalyticsDocumentApiService := analytics.NewNWDAFAnalyticsDocumentApiService()
	NWDAFAnalyticsDocumentApiController := analytics.NewNWDAFAnalyticsDocumentApiController(
		NWDAFAnalyticsDocumentApiService,
	)
	NWDAFContextDocumentApiService := analytics.NewNWDAFContextDocumentApiService()
	NWDAFContextDocumentApiController := analytics.NewNWDAFContextDocumentApiController(
		NWDAFContextDocumentApiService,
	)
	router := analytics.NewRouter(
		NWDAFAnalyticsDocumentApiController,
		NWDAFContextDocumentApiController,
	)
	server := &http.Server{
		Addr:         config.Server.Addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Server listening at %s", config.Server.Addr)
	log.Fatal(server.ListenAndServe())
}
