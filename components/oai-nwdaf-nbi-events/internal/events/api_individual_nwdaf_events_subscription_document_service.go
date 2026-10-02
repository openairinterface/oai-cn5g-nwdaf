/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * Functions of the events nbi service (delete, update subscriptions).
 */

package events

import (
	"context"
	"log"
	"net/http"
)

// IndividualNWDAFEventsSubscriptionDocumentApiService is a service that implements the logic for the IndividualNWDAFEventsSubscriptionDocumentApiServicer
// This service should implement the business logic for every endpoint for the IndividualNWDAFEventsSubscriptionDocumentApi API.
// Include any external packages or services that will be required by this service.
type IndividualNWDAFEventsSubscriptionDocumentApiService struct {
}

// NewIndividualNWDAFEventsSubscriptionDocumentApiService creates a default api service
func NewIndividualNWDAFEventsSubscriptionDocumentApiService() IndividualNWDAFEventsSubscriptionDocumentApiServicer {
	return &IndividualNWDAFEventsSubscriptionDocumentApiService{}
}

// DeleteNWDAFEventsSubscription - Delete an existing Individual NWDAF Events Subscription
func (s *IndividualNWDAFEventsSubscriptionDocumentApiService) DeleteNWDAFEventsSubscription(
	ctx context.Context,
	subscriptionId string,
) (ImplResponse, error) {
	subscriptionCh, ok := subscriptionTable[subscriptionId]
	// If the key exists
	if ok {
		close(subscriptionCh)
		delete(subscriptionTable, subscriptionId)
		delete(subscriptionData, subscriptionId)

		return Response(204, nil), nil
	}
	return Response(http.StatusBadRequest, ProblemDetails{}), nil
}

// UpdateNWDAFEventsSubscription - Update an existing Individual NWDAF Events Subscription
//
// Replaces the subscription in place, keeping its id (TS 29.520 §5.2.3): stop
// the goroutines watching the old criteria, then start new ones from the
// supplied body. Editing a threshold therefore does not change the URI the
// consumer already holds.
func (s *IndividualNWDAFEventsSubscriptionDocumentApiService) UpdateNWDAFEventsSubscription(
	ctx context.Context,
	subscriptionId string,
	nnwdafEventsSubscription NnwdafEventsSubscription,
) (ImplResponse, error) {
	oldCh, ok := subscriptionTable[subscriptionId]
	if !ok {
		return Response(http.StatusNotFound, ProblemDetails{}), nil
	}
	close(oldCh)

	subscriptionCh := make(chan string)
	subscriptionTable[subscriptionId] = subscriptionCh
	subscriptionData[subscriptionId] = nnwdafEventsSubscription
	for _, eventSub := range nnwdafEventsSubscription.EventSubscriptions {
		go handleSubscriptionEvent(ctx,
			eventSub,
			nnwdafEventsSubscription.NotificationURI,
			subscriptionId,
			subscriptionCh,
		)
	}
	log.Print("Updated subscription ", subscriptionId)
	return Response(200, NnwdafEventsSubscription{
		EventSubscriptions: nnwdafEventsSubscription.EventSubscriptions,
	}), nil
}

// GetNWDAFEventsSubscription - Read an existing Individual NWDAF Events Subscription.
//
// Not in TS 29.520, which offers only PUT and DELETE on this resource. Added
// because subscriptions live in memory: after an NWDAF restart a consumer's
// stored subscription id silently refers to nothing, and PUT -- the only other
// probe -- restarts the evaluation goroutine as a side effect, which resets
// threshold state. A plain read lets a consumer reconcile without disturbing it.
func (s *IndividualNWDAFEventsSubscriptionDocumentApiService) GetNWDAFEventsSubscription(
	ctx context.Context,
	subscriptionId string,
) (ImplResponse, error) {
	sub, ok := subscriptionData[subscriptionId]
	if !ok {
		return Response(http.StatusNotFound, ProblemDetails{}), nil
	}
	return Response(200, sub), nil
}
