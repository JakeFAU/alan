package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestPublicConsumer(t *testing.T) {
	var buf bytes.Buffer
	args := []string{"-term", "fork(relay(A),fork(B,B))", "-db", filepath.Join(t.TempDir(), "consumer.sqlite"),
		"-requests", `[{"id":2,"value":1,"mode":"fail"},{"id":2,"value":0,"mode":"absent"},{"id":2,"value":1,"mode":"answer"}]`}
	if err := execute(t.Context(), args, &buf); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Report runtime.RunReport
		Root   runtime.RunState
		Replay runtime.ReplayReport
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	d := got.Root.Projection.Data
	if !got.Report.Quiescent || len(got.Report.OpenRuns) != 0 || got.Report.RejectedAdmissions != 0 || got.Root.Projection.Status != "failed" ||
		d["decision"].Text != "diagnosed" || d["report_l_classification"].Text != "candidate_failed" || d["report_r_l_classification"].Text != "candidate_missing_reply" || d["report_r_r_classification"].Text != "answer" {
		t.Fatal("aggregate", got.Root)
	}
	row, err := d["report_r_r_answers"].At(0)
	if err != nil || row["id"].Number != 2 || row["value"].Number != 0 {
		t.Fatal("uncorrelated answer", row, err)
	}
	if got.Report.Roots[0].Charges.Invocations != 14 || got.Report.Roots[0].Charges.CreatedRuns != 14 {
		t.Fatal("physical spend", got.Report.Roots)
	}
	t.Logf("nested mixed standalone root=%s decision=%s steps=%d dispatches=%d charges=14", got.Root.ProgramID, d["decision"].Text, got.Report.Steps, got.Report.Dispatches)
}
