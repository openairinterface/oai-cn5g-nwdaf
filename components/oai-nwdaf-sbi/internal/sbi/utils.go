/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * This file contains utils functions.
 */

package sbi

import (
	"context"
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	amf_client "gitlab.eurecom.fr/oai/cn5g/oai-cn5g-nwdaf/components/oai-nwdaf-sbi/internal/amfclient"
	smf_client "gitlab.eurecom.fr/oai/cn5g/oai-cn5g-nwdaf/components/oai-nwdaf-sbi/internal/smfclient"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ------------------------------------------------------------------------------
// InitConfig - Initialize global variables (cfg and mongoClient) and subscribe to AMF and SMF
func InitConfig() {
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	clientOptions := options.Client().ApplyURI(config.Database.Uri)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connected to MongoDB.")
	mongoClient = client
	// Subscribe to all event notifications from AMF
	amfEventSubscription(
		config.Server.NotifUri+config.Amf.ApiRoute,
		config.Amf.NotifCorrId,
		config.Amf.NotifId,
	)
	// Subscribe to all event notifications from SMF
	smfEventSubscription(
		config.Server.NotifUri+config.Smf.ApiRoute,
		config.Smf.NotifId,
	)
}

// ------------------------------------------------------------------------------
func amfEventSubscription(
	amfEventNotifyUri string,
	amfNotifyCorrelationId string,
	amfNfId string,
) {
	// Store all AMF event types
	var amfEvents []amf_client.AmfEvent
	for _, amfEventTypeAnyOf := range amf_client.AllowedAmfEventTypeAnyOfEnumValues {
		amfEvents = append(amfEvents, *amf_client.NewAmfEvent(amfEventTypeAnyOf))
	}
	// Subscribe to all AMF event types
	amfCreateEventSubscription := *amf_client.NewAmfCreateEventSubscription(
		*amf_client.NewAmfEventSubscription(
			amfEvents,
			amfEventNotifyUri,
			amfNotifyCorrelationId,
			amfNfId,
		),
	)
	configuration := amf_client.NewConfiguration()
	amfApiClient := amf_client.NewAPIClient(configuration)
	resp, r, err := amfApiClient.SubscriptionsCollectionCollectionApi.CreateSubscription(
		context.Background()).AmfCreateEventSubscription(amfCreateEventSubscription).Execute()
	if err != nil {
		log.Printf(
			"Error when calling `SubscriptionsCollectionCollectionApi.CreateSubscription``: %v\n",
			err,
		)
		log.Printf("Full HTTP response: %v\n", r)
	}
	// response from `CreateSubscription`: AmfCreatedEventSubscription
	log.Printf(
		"Response from `SubscriptionsCollectionCollectionApi.CreateSubscription`: %v\n",
		resp,
	)
}

// ------------------------------------------------------------------------------
func smfEventSubscription(smfEventNotifyUri string, smfNfId string) {

	// Store all SMF event types
	var smfEventSubs []smf_client.EventSubscription
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_PDU_SES_EST),
	)
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_UE_IP_CH),
	)
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_PLMN_CH),
	)
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_DDDS),
	)
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_PDU_SES_REL),
	)
	smfEventSubs = append(smfEventSubs,
		*smf_client.NewEventSubscription(smf_client.SMFEVENTANYOF_QOS_MON),
	)
	// Subscribe to all SMF event types
	nsmfEventExposure := *smf_client.NewNsmfEventExposure(
		smfNfId,
		smfEventNotifyUri,
		smfEventSubs,
	)
	configuration := smf_client.NewConfiguration()
	smfApiClient := smf_client.NewAPIClient(configuration)
	resp, r, err := smfApiClient.SubscriptionsCollectionApi.CreateIndividualSubcription(
		context.Background()).NsmfEventExposure(nsmfEventExposure).Execute()
	if err != nil {
		log.Printf(
			"Error when calling `SubscriptionsCollectionApi.CreateIndividualSubcription``: %v\n",
			err,
		)
		log.Printf("Full HTTP response: %v\n", r)
	}
	// response from `CreateIndividualSubcription`: NsmfEventExposure
	log.Printf(
		"Response from `SubscriptionsCollectionApi.CreateIndividualSubcription`: %v\n",
		resp,
	)
}
