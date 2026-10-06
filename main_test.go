package main

import "testing"

func TestMakeDemoBatch(t *testing.T) {
	batch := makeDemoBatch(2, true)
	if len(batch) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(batch))
	}
	if batch[0]["Queue"] != "demo_queue" || batch[1]["Name"] != "PrintJob" {
		t.Fatalf("unexpected demo job: %#v", batch[0])
	}
	if batch[0]["IdempotencyKey"] != "job-idemp-0" {
		t.Fatalf("missing idempotency key: %#v", batch[0])
	}
}
