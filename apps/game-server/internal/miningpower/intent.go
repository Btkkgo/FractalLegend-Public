package miningpower

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

func ValidID(value string, max int) bool {
	if len(value) == 0 || len(value) > max {
		return false
	}
	for _, c := range value {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}
func ValidateIntent(intent ActionIntent) error {
	if !ValidID(intent.SourceEventID, 96) || !ValidID(intent.ActivityID, 128) || !ValidID(intent.ActivitySessionID, 128) || !ValidID(intent.BlockID, 128) || !ValidID(intent.BlockInstanceID, 128) || intent.ActivityID != "mpa:"+intent.SourceEventID {
		return ErrInvalidIntent
	}
	return nil
}
func forbiddenPower(key string) bool {
	switch strings.ToLower(strings.ReplaceAll(key, "_", "")) {
	case "miningpower", "power", "finalpower", "effectivepower", "participantpower", "blockpowershare", "rewardshare", "finalminingpower", "effectiveminingpower", "playershare", "totalminingpower":
		return true
	}
	return false
}
func DecodeActionIntent(payload []byte) (ActionIntent, error) {
	var intent ActionIntent
	if len(payload) > 4096 || !utf8.Valid(payload) {
		return intent, ErrInvalidIntent
	}
	d := json.NewDecoder(bytes.NewReader(payload))
	d.UseNumber()
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return intent, ErrInvalidIntent
	}
	seen := map[string]bool{}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return ActionIntent{}, ErrInvalidIntent
		}
		key, ok := token.(string)
		if !ok {
			return ActionIntent{}, ErrInvalidIntent
		}
		if forbiddenPower(key) {
			return ActionIntent{}, ErrClientPowerForbidden
		}
		if seen[key] {
			return ActionIntent{}, ErrInvalidIntent
		}
		seen[key] = true
		var target *string
		switch key {
		case "activityId":
			target = &intent.ActivityID
		case "sourceEventId":
			target = &intent.SourceEventID
		case "activitySessionId":
			target = &intent.ActivitySessionID
		case "blockId":
			target = &intent.BlockID
		case "blockInstanceId":
			target = &intent.BlockInstanceID
		default:
			return ActionIntent{}, ErrInvalidIntent
		}
		token, err = d.Token()
		if err != nil {
			return ActionIntent{}, ErrInvalidIntent
		}
		value, ok := token.(string)
		if !ok {
			return ActionIntent{}, ErrInvalidIntent
		}
		*target = value
	}
	token, err = d.Token()
	if err != nil || token != json.Delim('}') || len(seen) != 5 {
		return ActionIntent{}, ErrInvalidIntent
	}
	if _, err = d.Token(); err != io.EOF {
		return ActionIntent{}, ErrInvalidIntent
	}
	if err = ValidateIntent(intent); err != nil {
		return ActionIntent{}, err
	}
	return intent, nil
}
