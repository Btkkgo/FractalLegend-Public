package miningpower

import "testing"

// The legacy business ID is deliberately insufficient to authorize an instance.
func TestG20StableInstanceIntent(t *testing.T) {

	payload := []byte(`{"activityId":"mpa:event","sourceEventId":"event","activitySessionId":"session","blockId":"display","blockInstanceId":"mining-block-instance-11111111111111111111111111111111"}`)
	if _, err := DecodeActionIntent(payload); err != nil {
		t.Fatalf("stable identity rejected: %v", err)
	}
	if _, err := DecodeActionIntent([]byte(`{"activityId":"mpa:event","sourceEventId":"event","activitySessionId":"session","blockId":"display"}`)); err == nil {
		t.Fatal("legacy BlockID alone accepted")
	}
}

func TestG20InstanceProjectionIsolation(t *testing.T) {
	s := g20Snapshot()
	session := s.Sessions[0]
	session.ID = "session-2"
	session.BlockInstanceID = "instance-2"
	source := s.Sources[0]
	source.ID = "event-2"
	source.ActivityID = "mpa:event-2"
	source.ActivitySessionID = session.ID
	source.BlockInstanceID = "instance-2"
	activity := s.Activities[0]
	activity.ActivityID = source.ActivityID
	activity.SourceEventID = source.ID
	activity.ActivitySessionID = session.ID
	activity.BlockInstanceID = "instance-2"
	activity.ValidationSnapshot.BlockInstanceID = "instance-2"
	participant := s.Participants[0]
	participant.ActivitySessionID = session.ID
	participant.BlockInstanceID = "instance-2"
	block := s.Blocks[0]
	block.BlockInstanceID = "instance-2"
	s.Sessions = append(s.Sessions, session)
	s.Sources = append(s.Sources, source)
	s.Activities = append(s.Activities, activity)
	s.Participants = append(s.Participants, participant)
	s.Blocks = append(s.Blocks, block)
	r, err := ReconcileSnapshot(s)
	if err != nil || r.Status != "PASS" {
		t.Fatalf("same business ID combined concrete instances: %+v %v", r, err)
	}
	rebuilt, err := RebuildParticipants(s)
	if err != nil || len(rebuilt.Participants) != 2 || rebuilt.Participants[0].BlockInstanceID == rebuilt.Participants[1].BlockInstanceID {
		t.Fatalf("instance projection=%+v %v", rebuilt, err)
	}
}

func TestG20ValidationSnapshotIdentity(t *testing.T) {
	s := g20Snapshot()
	s.Activities[0].ValidationSnapshot.BlockInstanceID = "recreated-other-instance"
	r, err := ReconcileSnapshot(s)
	if err != nil || r.Status == "PASS" {
		t.Fatalf("historical validation identity mismatch ignored: %+v %v", r, err)
	}
}
