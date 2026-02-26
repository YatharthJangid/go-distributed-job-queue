package lib

import (
	"testing"
)

func TestMsgpackRoundTrip(t *testing.T) {
	j := &Job{
		ID:          "id-1",
		Name:        "PrintJob",
		Queue:       "demo_queue",
		Args:        map[string]interface{}{"id": float64(1)},
		Retry:       true,
		EnqueueTime: 1730663980.0,
	}
	b, err := j.ToBytes()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	j2, err := FromBytes(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if j2.Name != j.Name || j2.Queue != j.Queue {
		t.Fatalf("mismatch: %+v vs %+v", j2, j)
	}
}

func TestPoolReuseClearsFields(t *testing.T) {
	j := GetJob()
	j.ID, j.Name, j.Queue = "X", "Y", "Z"
	j.Args["k"] = float64(1)
	PutJob(j)

	j2 := GetJob()
	defer PutJob(j2)
	if j2.ID != "" || j2.Name != "" || j2.Queue != "" {
		t.Fatalf("expected cleared fields, got %+v", j2)
	}
	if _, ok := j2.Args["k"]; ok {
		t.Fatalf("expected cleared Args map")
	}
}

func TestJobValidateErrors(t *testing.T) {
	tests := []struct {
		job *Job
		err string
	}{
		{&Job{Name: "", Queue: "q"}, "name/queue empty"},
		{&Job{Name: "n", Queue: ""}, "name/queue empty"},
	}
	for _, tt := range tests {
		if err := tt.job.Validate(); err == nil {
			t.Fatalf("expected error for %+v", tt.job)
		}
	}
}

func TestFromBytesInvalidData(t *testing.T) {
	_, err := FromBytes([]byte("invalid msgpack"))
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}
