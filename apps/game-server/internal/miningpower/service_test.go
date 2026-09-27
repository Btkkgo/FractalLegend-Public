package miningpower

import (
	"context"
	"testing"
)

func TestG20ProductionClosed(t *testing.T) {
	result, err := NewService(nil, DevelopmentRuleVersion, true).Validate(context.Background(), Principal{AccountID: "account-1", PlayerID: "player-1"}, ActionIntent{ActivityID: "mpa:event-1", SourceEventID: "event-1", ActivitySessionID: "session-1", BlockID: "block-1", BlockInstanceID: "instance-1"})
	if err != nil || result.Status != StatusNotEligible || result.ReasonCode != "PRODUCTION_NOT_APPROVED" || result.AppliedPower != 0 {
		t.Fatalf("T90 production boundary=%+v error=%v", result, err)
	}
}
