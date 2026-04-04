/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * This file contains data structures and global variables.
 */

package engine

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Global variable
var mongoClient *mongo.Client
var config EngineConfig

// ------------------------------------------------------------------------------
// Type of EngineConfig structure
type EngineConfig struct {
	Routes struct {
		NumOfUe       string `envconfig:"ENGINE_NUM_OF_UE_ROUTE"`
		SessSuccRatio string `envconfig:"ENGINE_SESS_SUCC_RATIO_ROUTE"`
		UeComm        string `envconfig:"ENGINE_UE_COMMUNICATION_ROUTE"`
		UeMob         string `envconfig:"ENGINE_UE_MOBILITY_ROUTE"`
	}
	Database struct {
		Uri               string `envconfig:"MONGODB_URI"`
		DbName            string `envconfig:"MONGODB_DATABASE_NAME"`
		CollectionAmfName string `envconfig:"MONGODB_COLLECTION_NAME_AMF"`
		CollectionSmfName string `envconfig:"MONGODB_COLLECTION_NAME_SMF"`
	}
}

// ------------------------------------------------------------------------------
// Type of network_performance data to request engine
type EngineReqData struct {
	StartTs time.Time `json:"startTs,omitempty"`
	EndTs   time.Time `json:"endTs,omitempty"`
	Tais    []Tai     `json:"tais,omitempty"`
	Dnns    []string  `json:"dnns,omitempty"`
	Snssaia []Snssai  `json:"snssaia,omitempty"`
	Supi    string    `json:"supi,omitempty"`
}

type Tai struct {
	PlmnId PlmnId `json:"plmnId"`
	Tac    string `json:"tac"`
	Nid    string `json:"nid,omitempty"`
}

type PlmnId struct {
	Mcc string `json:"mcc"`
	Mnc string `json:"mnc"`
}

type Snssai struct {
	Sst int32  `json:"sst"`
	Sd  string `json:"sd,omitempty"`
}

// ------------------------------------------------------------------------------
// Type of network_performance response from engine.
type NwPerfResp struct {
	RelativeRatio int32 `json:"relativeRatio,omitempty"`
	AbsoluteNum   int32 `json:"absoluteNum,omitempty"`
	Confidence    int32 `json:"confidence,omitempty"`
}

// ------------------------------------------------------------------------------
// Type of Ue_communication response from engine.
type UeCommResp struct {
	CommDur       int32     `json:"commDur"`
	Ts            time.Time `json:"ts,omitempty"`
	UlVol         int64     `json:"ulVol,omitempty"`
	UlVolVariance float32   `json:"ulVolVariance,omitempty"`
	DlVol         int64     `json:"dlVol,omitempty"`
	DlVolVariance float32   `json:"dlVolVariance,omitempty"`
}
