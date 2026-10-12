/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * This is main file of the oai-nwdaf-nbi-events HTTP server.
 */

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openairinterface/oai-cn5g-nwdaf/components/oai-nwdaf-nbi-events/internal/events"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	events.InitConfig()
	IndividualNWDAFEventSubscriptionTransferDocumentApiService := events.NewIndividualNWDAFEventSubscriptionTransferDocumentApiService()
	IndividualNWDAFEventSubscriptionTransferDocumentApiController := events.NewIndividualNWDAFEventSubscriptionTransferDocumentApiController(
		IndividualNWDAFEventSubscriptionTransferDocumentApiService,
	)
	IndividualNWDAFEventsSubscriptionDocumentApiService := events.NewIndividualNWDAFEventsSubscriptionDocumentApiService()
	IndividualNWDAFEventsSubscriptionDocumentApiController := events.NewIndividualNWDAFEventsSubscriptionDocumentApiController(
		IndividualNWDAFEventsSubscriptionDocumentApiService,
	)
	NWDAFEventSubscriptionTransfersCollectionApiService := events.NewNWDAFEventSubscriptionTransfersCollectionApiService()
	NWDAFEventSubscriptionTransfersCollectionApiController := events.NewNWDAFEventSubscriptionTransfersCollectionApiController(
		NWDAFEventSubscriptionTransfersCollectionApiService,
	)
	NWDAFEventsSubscriptionsCollectionApiService := events.NewNWDAFEventsSubscriptionsCollectionApiService()
	NWDAFEventsSubscriptionsCollectionApiController := events.NewNWDAFEventsSubscriptionsCollectionApiController(
		NWDAFEventsSubscriptionsCollectionApiService,
	)
	router := events.NewRouter(
		IndividualNWDAFEventSubscriptionTransferDocumentApiController,
		IndividualNWDAFEventsSubscriptionDocumentApiController,
		NWDAFEventSubscriptionTransfersCollectionApiController,
		NWDAFEventsSubscriptionsCollectionApiController,
	)
	server := &http.Server{
		Addr:         config.Server.Addr,
		// HTTP/1.1 for the NBI gateway and AFs, h2c for SBI consumers (NEF)
		Handler:      h2c.NewHandler(router, &http2.Server{}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Server listening at %s", config.Server.Addr)
	log.Fatal(server.ListenAndServe())
}
