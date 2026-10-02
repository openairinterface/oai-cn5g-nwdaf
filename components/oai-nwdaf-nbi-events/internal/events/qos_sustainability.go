/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * QOS_SUSTAINABILITY event (3GPP TS 23.288 §6.9, TS 29.520).
 *
 * The consumer subscribes with ranUeThrouThds, e.g. ["5 Mbps"]. We ask the
 * engine for the throughput each UE is actually getting (derived from the UPF's
 * usage reports) and report the QoS as not sustained when it sits below that.
 *
 * With notificationMethod THRESHOLD the notification is sent on the TRANSITION
 * into degradation, not on every poll: "the event is produced when the QoS
 * starts degrading". Recovery is reported once too, so a consumer can clear it.
 */

package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Mirrors the engine's QosSustainResp.
type QosSustainUe struct {
	Supi           string `json:"supi,omitempty"`
	UeIpv4         string `json:"ueIpv4,omitempty"`
	Reports        int32  `json:"reports"`
	LastTs         int64  `json:"lastTs,omitempty"`
	LastDlThrouBps int64  `json:"lastDlThrouBps"`
	LastUlThrouBps int64  `json:"lastUlThrouBps"`
	AvgDlThrouBps  int64  `json:"avgDlThrouBps"`
	AvgUlThrouBps  int64  `json:"avgUlThrouBps"`
}

type QosSustainResp struct {
	Ts            time.Time      `json:"ts"`
	Ues           []QosSustainUe `json:"ues,omitempty"`
	MinDlThrouBps int64          `json:"minDlThrouBps"`
	MinUlThrouBps int64          `json:"minUlThrouBps"`
	Confidence    int32          `json:"confidence"`
}

// ------------------------------------------------------------------------------
// parseBitRate - "5 Mbps" -> 5000000. TS 29.571 BitRate: decimal value plus an
// SI-prefixed unit, x1000 multipliers, "K" for kilo.
func parseBitRate(s string) (int64, error) {
	t := strings.TrimSpace(s)
	units := []struct {
		suffix string
		mult   float64
	}{
		{"Tbps", 1e12}, {"Gbps", 1e9}, {"Mbps", 1e6}, {"Kbps", 1e3}, {"bps", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(t, u.suffix) {
			num := strings.TrimSpace(strings.TrimSuffix(t, u.suffix))
			v, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0, err
			}
			return int64(v * u.mult), nil
		}
	}
	return 0, errors.New("unrecognised bit rate: " + s)
}

// ------------------------------------------------------------------------------
// getQosSustainNotifData - build the QosSustainabilityInfo list for a
// subscription, one entry per configured ranUeThrouThd.
func getQosSustainNotifData(eventSub EventSubscription) ([]QosSustainabilityInfo, error) {
	log.Printf("Getting QoS Sustainability Notification Data")
	var infos []QosSustainabilityInfo
	resp, err := requestQosSustainEngine(eventSub, config.Engine.Uri+config.Routes.QosSustain)
	if err != nil {
		return infos, err
	}
	thds := eventSub.RanUeThrouThds
	if len(thds) == 0 {
		// No threshold given: still report what was observed, so a PERIODIC
		// consumer can chart it.
		thds = []string{""}
	}
	for _, thd := range thds {
		info := QosSustainabilityInfo{
			StartTs:       eventSub.ExtraReportReq.StartTs,
			EndTs:         eventSub.ExtraReportReq.EndTs,
			RanUeThrouThd: thd,
			Confidence:    resp.Confidence,
		}
		if len(eventSub.Snssaia) > 0 {
			info.Snssai = eventSub.Snssaia[0]
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// recoveryMargin - how far back above the threshold a UE must climb before the
// QoS counts as sustained again. Without it a UE parked exactly on its
// threshold (the usual case: the threshold is set to the enforced MBR) flaps
// between crossed and recovered on every report, because successive reports
// land a few kbit/s either side.
const recoveryMargin = 1.10

// ------------------------------------------------------------------------------
// qosSustainDegraded - which UEs are currently below a configured threshold?
//
// Keyed per UE rather than a single verdict: with one global flag, a second UE
// degrading while a first is already down produces no notification at all,
// because the overall state never changes.
//
// prev carries the UEs that were degraded at the last evaluation, so the
// comparison can be asymmetric -- a UE already down has to climb back above
// threshold*recoveryMargin to count as recovered.
func qosSustainDegraded(
	eventSub EventSubscription,
	prev map[string]bool,
) (map[string]bool, string, error) {
	now := make(map[string]bool)
	resp, err := requestQosSustainEngine(eventSub, config.Engine.Uri+config.Routes.QosSustain)
	if err != nil {
		return now, "", err
	}
	reason := ""
	for _, thdStr := range eventSub.RanUeThrouThds {
		thd, err := parseBitRate(thdStr)
		if err != nil {
			log.Print("QOS_SUSTAINABILITY: ", err)
			continue
		}
		for _, ue := range resp.Ues {
			// Only judge a UE that has reported at least twice: a single report
			// covers a partly-idle interval and reads artificially low.
			if ue.Reports < 2 {
				continue
			}
			limit := thd
			if prev[ue.Supi] {
				limit = int64(float64(thd) * recoveryMargin)
			}
			if ue.LastDlThrouBps < limit {
				now[ue.Supi] = true
				if !prev[ue.Supi] {
					reason = ue.Supi + " (" + ue.UeIpv4 + ") downlink " +
						strconv.FormatInt(ue.LastDlThrouBps, 10) + " bps is below " + thdStr
				}
			}
		}
	}
	if reason == "" {
		for supi := range prev {
			if !now[supi] {
				reason = supi + " is back above its threshold"
				break
			}
		}
	}
	return now, reason, nil
}

// ------------------------------------------------------------------------------
func requestQosSustainEngine(
	eventSub EventSubscription,
	enginePath string,
) (QosSustainResp, error) {
	var out QosSustainResp
	var engineReqData EngineReqData
	engineReqData.StartTs = eventSub.ExtraReportReq.StartTs
	engineReqData.EndTs = eventSub.ExtraReportReq.EndTs
	if len(eventSub.TgtUe.Supis) > 0 {
		engineReqData.Supi = eventSub.TgtUe.Supis[0]
	}
	reqJson, err := json.Marshal(engineReqData)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest(http.MethodGet, enginePath, bytes.NewBuffer(reqJson))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return out, err
	}
	return out, nil
}
