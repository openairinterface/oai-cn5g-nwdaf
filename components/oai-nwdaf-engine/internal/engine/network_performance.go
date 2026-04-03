/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * This file contains functions related to Network Performance event ID.
 */

package engine

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
)

// ------------------------------------------------------------------------------
// nwPerfNumOfUe - get the number of Ue.
func nwPerfNumOfUe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":
		log.Printf("Getting Number of UE Info from DB")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		var engineReqData EngineReqData
		err = json.Unmarshal(body, &engineReqData)
		if err != nil {
			http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
			return
		}
		// Create filter and calculate number of UEs
		filter := getFilterNwPerfNumUe(engineReqData)
		db := mongoClient.Database(config.Database.DbName)
		collection := db.Collection(config.Database.CollectionAmfName)
		log.Printf("Counting documents from mongo DB using filter ...")
		absoluteNum, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			http.Error(w, "Error counting documents", http.StatusInternalServerError)
			return
		}
		//TODO - Implement relative ratio and confidence
		relativeRatio, confidence := int32(0), int32(0)
		// Prepare http response body
		nwPerfResp := NwPerfResp{
			RelativeRatio: relativeRatio,
			AbsoluteNum:   int32(absoluteNum),
			Confidence:    confidence,
		}
		w.Header().Set("Content-Type", "application/json")
		jsonResp, err := json.Marshal(nwPerfResp)
		if err != nil {
			http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
			return
		}
		w.Write(jsonResp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ------------------------------------------------------------------------------
// nwPerfNumOfPdu - Get the number of PDU sessions.
func nwPerfNumOfPdu(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":
		log.Printf("Getting Number of Pdu Sessions from DB")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		var engineReqData EngineReqData
		err = json.Unmarshal(body, &engineReqData)
		if err != nil {
			http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
			return
		}
		// get filter to calculate number of users in given network area
		filter := getFilterNwPerfNumPdu(engineReqData)
		db := mongoClient.Database(config.Database.DbName)
		collection := db.Collection(config.Database.CollectionSmfName)
		log.Printf("Counting documents from mongo DB using filter ...")
		absoluteNum, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			http.Error(w, "Error counting documents", http.StatusInternalServerError)
			return
		}
		//Implement relative ratio and confidence
		relativeRatio, confidence := int32(0), int32(0)
		// prepare http response body
		nwPerfResp := NwPerfResp{
			RelativeRatio: relativeRatio,
			AbsoluteNum:   int32(absoluteNum),
			Confidence:    confidence,
		}
		w.Header().Set("Content-Type", "application/json")
		jsonResp, err := json.Marshal(nwPerfResp)
		if err != nil {
			http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
			return
		}
		w.Write(jsonResp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ------------------------------------------------------------------------------
// getFilterNwPerfNumUe - Get request filter that will be used to calculates the number of UEs
func getFilterNwPerfNumUe(engineReqData EngineReqData) bson.D {
	log.Printf("Constructing filter for DB search ...")
	// get startTs and endTs
	startTs, endTs := getExtraReportReq(engineReqData)
	timeStampCondition := getTimeStampCondition(startTs, endTs)
	// get timestamps filter - UE that are registered at timestampCondition
	filterTimeReg := bson.D{
		{"rminfolist",
			bson.M{"$elemMatch": bson.M{"rminfo.rmstate.rmstateanyof": "REGISTERED",
				"timestamp": timeStampCondition}},
		}}
	// get Network Area filter
	filterLoc := bson.D{}
	if engineReqData.Tais != nil {
		tacs := make([]string, 0)
		plmnids := make([]PlmnId, 0)
		for _, t := range engineReqData.Tais {
			tacs = append(tacs, t.Tac)
			plmnids = append(plmnids, t.PlmnId)
		}
		// filter UE that are located in network area at timestampCondition
		filterLoc = bson.D{{"locationlist",
			bson.M{"$elemMatch": bson.M{
				"userlocation.nrlocation.tai.tac":    bson.D{{"$in", tacs}},
				"userlocation.nrlocation.tai.plmnid": bson.D{{"$in", plmnids}},
				"timestamp":                          timeStampCondition,
			}}},
		}
	}
	// combien the two filters
	filter := bson.D{{"$and", bson.A{filterTimeReg, filterLoc}}}
	return filter
}

// ------------------------------------------------------------------------------
// getFilterNwPerfNumPdu - Get filter that will be used to calculates the number PduSessionEst
func getFilterNwPerfNumPdu(engineReqData EngineReqData) bson.D {
	log.Printf("Constructing PDU session ratio filter for DB search ...")
	// get startTs and endTs
	startTs, endTs := getExtraReportReq(engineReqData)
	timeStampCondition := getTimeStampCondition(startTs, endTs)
	// get timestamps filter
	filterTimePdu := bson.D{{"pdusesestlist",
		bson.M{"$elemMatch": bson.M{
			"timestamp": timeStampCondition,
		}},
	},
	}
	// get Dnn filter
	filterDnn := bson.D{}
	if engineReqData.Dnns != nil {
		// filter UE that have dnn in dnns
		filterDnn = bson.D{{"pdusesestlist",
			bson.M{"$elemMatch": bson.M{
				"dnn": bson.D{{"$in", engineReqData.Dnns}},
			}},
		}}
	}
	// get Snssai filter
	filterSnssai := bson.D{}
	if engineReqData.Snssaia != nil {
		// filter UE that have Snssai in dSnssai
		filterSnssai = bson.D{{"pdusesestlist",
			bson.M{"$elemMatch": bson.M{
				"snssai": bson.D{{"$in", engineReqData.Snssaia}},
			}},
		}}
	}
	// combien the three filters
	filter := bson.D{{"$and", bson.A{filterTimePdu, filterDnn, filterSnssai}}}
	return filter
}
