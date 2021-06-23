package models

import (
	"encoding/json"
	"time"
)

const (
	MessageTypeUserOnline      = "MessageTypeUserOnline"
	MessageTypeUserOffline     = "MessageTypeUserOffline"
	MessageTypeOilFieldOnline  = "MessageTypeOilFieldOnline"
	MessageTypeOilFieldOffline = "MessageTypeOilFieldOffline"
	MessageTypeAlarm           = "MessageTypeAlarm"

	MessageTypeCloudSyncGzip    = "MessageTypeCloudSyncGZIP"
	MessageTypeCloudSyncGzipAck = "MessageTypeCloudSyncGzipAck"
)

type InputMessage struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

type OutputMessage struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Body      json.RawMessage `json:"body"`
}

func (om *OutputMessage) MarshalJSON() ([]byte, error) {
	type Alias OutputMessage
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: om.Timestamp.Unix(),
		Alias:     (*Alias)(om),
	})
}
