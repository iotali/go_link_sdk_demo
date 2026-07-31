package mqtt

import (
	"testing"
	"time"
)

func TestBuildEventReportMessageUsesThingModelEventShape(t *testing.T) {
	now := time.Unix(123, 456000000).UTC()
	msg, eventName, err := buildEventReportMessage(map[string]interface{}{
		"event_type": "overheat_alarm",
		"timestamp":  int64(1_708_934_400),
		"data": map[string]interface{}{
			"temperature": 85.5,
			"threshold":   80,
		},
	}, now)
	if err != nil {
		t.Fatalf("build event message: %v", err)
	}
	if eventName != "overheat_alarm" {
		t.Fatalf("eventName = %s", eventName)
	}
	if msg["id"] != "123456" {
		t.Fatalf("id = %v", msg["id"])
	}
	if _, ok := msg["method"]; ok {
		t.Fatalf("unexpected method = %v", msg["method"])
	}
	params, ok := msg["params"].(map[string]interface{})
	if !ok {
		t.Fatalf("params type = %T", msg["params"])
	}
	if _, ok := params["eventType"]; ok {
		t.Fatalf("unexpected eventType = %v", params["eventType"])
	}
	rawEvent, ok := params["overheat_alarm"].(map[string]interface{})
	if !ok {
		t.Fatalf("event param type = %T", params["overheat_alarm"])
	}
	if rawEvent["time"] != int64(1_708_934_400_000) {
		t.Fatalf("time = %v", rawEvent["time"])
	}
	value, ok := rawEvent["value"].(map[string]interface{})
	if !ok {
		t.Fatalf("event value type = %T", rawEvent["value"])
	}
	if value["temperature"] != 85.5 {
		t.Fatalf("temperature = %v", value["temperature"])
	}
}

func TestBuildEventReportMessageRequiresEventType(t *testing.T) {
	_, _, err := buildEventReportMessage(map[string]interface{}{}, time.Unix(123, 0).UTC())
	if err == nil {
		t.Fatal("missing error")
	}
}
