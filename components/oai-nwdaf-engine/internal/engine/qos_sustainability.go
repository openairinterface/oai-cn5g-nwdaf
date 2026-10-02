/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * QoS Sustainability analytics (3GPP TS 23.288 §6.9, TS 29.520 §6.1.3).
 *
 * The spec allows two ways to express "QoS is no longer sustained": QoS flow
 * retainability (qosFlowRetThd) and RAN UE throughput (ranUeThrouThd). We serve
 * the throughput one, because it is derivable from data the core already
 * produces: the UPF's URR usage reports, forwarded by the SMF as QOS_MON event
 * exposure and stored by the NWDAF SBI.
 *
 * Each usage report carries the volume since the PREVIOUS report, and duration
 * is cumulative session time -- so the interval between two reports gives the
 * throughput for that interval directly.
 */

package engine

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ------------------------------------------------------------------------------
// qosSustain - per-UE observed throughput, for comparison against ranUeThrouThd.
func qosSustain(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("Getting QoS Sustainability from DB")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	var engineReqData EngineReqData
	if err = json.Unmarshal(body, &engineReqData); err != nil {
		http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
		return
	}

	collection := mongoClient.Database(config.Database.DbName).
		Collection(config.Database.CollectionSmfName)
	cursor, err := collection.Find(context.Background(), GetFilterUeComm(engineReqData))
	if err != nil {
		http.Error(w, "Error finding documents", http.StatusInternalServerError)
		return
	}
	startTs, endTs := getExtraReportReq(engineReqData)
	timeStamp := calculateTimeStamp(startTs, endTs)

	resp := QosSustainResp{Ts: time.Now(), MinDlThrouBps: -1, MinUlThrouBps: -1}
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			http.Error(w, "Error decoding document", http.StatusInternalServerError)
			return
		}
		supi, _ := doc["_id"].(string)
		if engineReqData.Supi != "" && engineReqData.Supi != supi {
			continue
		}
		ue := QosSustainUe{Supi: supi, UeIpv4: ueIpv4FromDoc(doc)}

		qosMonList, ok := doc["qosmonlist"].(primitive.A)
		if !ok {
			continue
		}
		// Reports are stored in arrival order; walk them keeping the previous
		// cumulative duration so each interval is known.
		prevDur := int32(-1)
		var sumUlBits, sumDlBits, sumSecs int64
		for _, elem := range qosMonList {
			qm, ok := elem.(bson.M)
			if !ok {
				continue
			}
			ts, _ := qm["timestamp"].(int64)
			if !matchTimeStamp(ts, timeStamp, startTs, endTs) {
				continue
			}
			cd, ok := qm["customized_data"].(bson.M)
			if !ok {
				continue
			}
			ur, ok := cd["usagereport"].(bson.M)
			if !ok {
				continue
			}
			vol, ok := ur["volume"].(bson.M)
			if !ok {
				continue
			}
			ul, _ := vol["uplink"].(int64)
			dl, _ := vol["downlink"].(int64)
			dur, _ := ur["duration"].(int32)

			interval := int64(30) // the UPF's reporting period
			if prevDur >= 0 && dur > prevDur {
				interval = int64(dur - prevDur)
			}
			prevDur = dur
			if interval <= 0 {
				continue
			}
			ue.Reports++
			ue.LastDlThrouBps = dl * 8 / interval
			ue.LastUlThrouBps = ul * 8 / interval
			ue.LastTs = ts
			sumDlBits += dl * 8
			sumUlBits += ul * 8
			sumSecs += interval
		}
		if ue.Reports == 0 || sumSecs == 0 {
			continue // an idle session produces no usage report at all
		}
		ue.AvgDlThrouBps = sumDlBits / sumSecs
		ue.AvgUlThrouBps = sumUlBits / sumSecs
		if resp.MinDlThrouBps < 0 || ue.LastDlThrouBps < resp.MinDlThrouBps {
			resp.MinDlThrouBps = ue.LastDlThrouBps
		}
		if resp.MinUlThrouBps < 0 || ue.LastUlThrouBps < resp.MinUlThrouBps {
			resp.MinUlThrouBps = ue.LastUlThrouBps
		}
		resp.Ues = append(resp.Ues, ue)
	}
	if resp.MinDlThrouBps < 0 {
		resp.MinDlThrouBps = 0
	}
	if resp.MinUlThrouBps < 0 {
		resp.MinUlThrouBps = 0
	}
	// Confidence is a plain sample count here: one usage report is a weak basis
	// for saying QoS is degrading, a dozen is a reasonable one.
	for _, u := range resp.Ues {
		if u.Reports > resp.Confidence {
			resp.Confidence = u.Reports
		}
	}

	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}
	w.Write(jsonResp)
}

// ------------------------------------------------------------------------------
// ueIpv4FromDoc - the UE address the SMF reported, for correlating with the AF.
func ueIpv4FromDoc(doc bson.M) string {
	for _, key := range []string{"pdusesestlist", "ueipchlist"} {
		list, ok := doc[key].(primitive.A)
		if !ok {
			continue
		}
		for _, elem := range list {
			if entry, ok := elem.(bson.M); ok {
				if ip, ok := entry["adipv4addr"].(string); ok && ip != "" {
					return ip
				}
			}
		}
	}
	return ""
}
