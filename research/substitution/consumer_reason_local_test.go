package substitution

import (
	"fmt"
	"testing"
)

// Local maximum fixtures are specification assertions, not explored graphs or
// replacement acceptance. Qualification of the original graphs remains gated.
func reasonLocalMaximumFixture(mode string, v Variant) (Config, State) {
	join := JoinConfig{Request: Request{ID: 2, Value: 1, Mode: mode}, I: 3, R: 3, Depth: 2}
	c := ConsumerReasonLocalConfig(join)
	c.Closure = false
	s := Initial(c)
	s.I, s.R, s.PeerTicket, s.RootPC = 2, 2, spent, terminal
	e := &s.Ops[entry]
	e.Ticket, e.Admission, e.RequestStage, e.Choice = spent, 1, 2, 1
	e.PC, e.Freeze, e.Reply = terminal, success, 2
	e.Published, e.Delivered, e.Consumed = true, true, true
	e.Status, e.Reason, e.Present, e.ID = "succeeded", "member_succeeded", true, c.Request.ID
	s.RootStatus = "succeeded"
	s.Ops[helper].Ticket = excluded
	if v == Helper {
		s.I, s.R, e.Choice = 3, 3, 2
		h := &s.Ops[helper]
		h.Ticket, h.Admission, h.RequestStage, h.PC, h.Freeze, h.Reply = spent, 1, 2, terminal, success, 2
		h.Published, h.Delivered, h.Consumed, h.Present = true, true, true, true
		h.Status, h.Reason, h.ID = "succeeded", "member_succeeded", c.Request.ID
	}
	if mode == "fail" || mode == "absent" {
		e.Status, e.Reason, e.Reply, e.Present, e.ID = "failed", "missing_reply", 0, false, 0
		s.RootStatus = "failed"
		if mode == "fail" {
			e.Freeze, e.Reason = failure, "member_failed"
		}
	}
	if mode == "pending" {
		e.PC, e.Freeze, e.Reply, e.ID = pending, none, 0, 0
		e.Published, e.Delivered, e.Consumed, e.Present = false, false, false, false
		e.Status, e.Reason = "", ""
		s.RootPC, s.RootStatus = await, ""
	}
	return c, s
}

func TestConsumerReasonLocalModeProgress(t *testing.T) {
	for _, mode := range []string{"answer", "fail", "absent", "pending"} {
		for _, variant := range []Variant{Local, Helper} {
			t.Run(fmt.Sprintf("%s/%s", mode, variant), func(t *testing.T) {
				c, s := reasonLocalMaximumFixture(mode, variant)
				if err := CheckConsumerReasonLocalMaximum(c, variant, s); err != nil {
					t.Fatalf("specified local maximum: %v", err)
				}
				// Finite safety-prefix acceptance is insufficient for every mode.
				prefix := Initial(c)
				if err := CheckConsumerReasonLocalMaximum(c, variant, prefix); err == nil {
					t.Fatal("initial safe prefix accepted as mode progress")
				}
				if variant == Helper {
					bad := s
					bad.Ops[helper].Consumed = false
					if err := CheckConsumerReasonLocalMaximum(c, variant, bad); err == nil {
						t.Fatal("candidate progress accepted with original helper unconsumed")
					}
				}
				bad := s
				bad.PeerTicket = possible
				if err := CheckConsumerReasonLocalMaximum(c, variant, bad); err == nil {
					t.Fatal("maximum omitted continuously enabled q service")
				}
				bad = s
				if mode == "pending" {
					bad.Ops[entry].PC = receive
				} else {
					bad.Ops[entry].Reason = "wrong_reason"
				}
				if err := CheckConsumerReasonLocalMaximum(c, variant, bad); err == nil {
					t.Fatal("wrong reason/designated receive accepted")
				}
			})
		}
	}
}

func TestConsumerReasonLocalRefusalIsNotModeProgress(t *testing.T) {
	c, s := reasonLocalMaximumFixture("fail", Helper)
	s.Ops[entry].FailureReady = true
	if err := CheckConsumerReasonLocalMaximum(c, Helper, s); err == nil {
		t.Fatal("helper refusal silently accepted as entitled requested failure")
	}
	c.Closure = true
	if err := CheckConsumerReasonLocalMaximum(c, Helper, s); err == nil {
		t.Fatal("closure relation configuration accepted as open progress")
	}
}
