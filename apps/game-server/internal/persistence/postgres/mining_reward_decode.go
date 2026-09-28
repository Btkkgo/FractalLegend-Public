package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"

	"fractallegend/game-server/internal/miningpower"
)

const miningRewardMaxCommandBytes = 4096

// DecodeMiningRewardCommandTEST is the only JSON boundary for the internal
// TEST settlement adapter. In particular, callers cannot supply R, weights,
// beneficiary, quantities, or any other economic authority.
func DecodeMiningRewardCommandTEST(raw []byte) (MiningRewardSettlementCommand, error) {
	var command MiningRewardSettlementCommand
	if len(raw) == 0 || len(raw) > miningRewardMaxCommandBytes {
		return command, miningpower.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return command, miningpower.ErrInvalidInput
	}
	seen := make(map[string]bool, 3)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil || len(value) < 2 || value[0] != '"' {
			return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
		}
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
		}
		switch key {
		case "commandID":
			command.CommandID = decoded
		case "blockInstanceID":
			command.BlockInstanceID = decoded
		case "expectedSealDigest":
			command.ExpectedSealDigest = decoded
		default:
			return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
	}
	if _, err := decoder.Token(); err != io.EOF {
		return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
	}
	const prefix = "mining-block-instance-"
	instance := strings.TrimPrefix(command.BlockInstanceID, prefix)
	_, instanceErr := hex.DecodeString(instance)
	_, digestErr := hex.DecodeString(command.ExpectedSealDigest)
	if len(seen) != 3 || !validMiningBlockID(command.CommandID) ||
		!strings.HasPrefix(command.BlockInstanceID, prefix) || len(instance) != 32 || instanceErr != nil ||
		len(command.ExpectedSealDigest) != 64 || digestErr != nil ||
		strings.ToLower(command.BlockInstanceID) != command.BlockInstanceID ||
		strings.ToLower(command.ExpectedSealDigest) != command.ExpectedSealDigest {
		return MiningRewardSettlementCommand{}, miningpower.ErrInvalidInput
	}
	return command, nil
}

// SettleMiningRewardPayloadTEST preserves the same TEST database guard as the
// typed coordinator after strict decode. There is no production/client route.
func (s *Store) SettleMiningRewardPayloadTEST(ctx context.Context, raw []byte) (MiningRewardSettlementReceipt, error) {
	command, err := DecodeMiningRewardCommandTEST(raw)
	if err != nil {
		return MiningRewardSettlementReceipt{}, err
	}
	return s.SettleMiningRewardTEST(ctx, command)
}
