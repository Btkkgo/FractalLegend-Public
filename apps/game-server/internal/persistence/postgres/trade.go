package postgres

import (
	"context"
	"errors"

	"fractallegend/game-server/internal/persistence"
	"fractallegend/game-server/internal/trade"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type tradeTransaction struct {
	ctx     context.Context
	tx      pgx.Tx
	store   *Store
	session *trade.Session
	pending map[string]persistence.CharacterAggregate
}

func (t *tradeTransaction) Session() *trade.Session { return t.session }

func (t *tradeTransaction) LoadCharacter(id string) (persistence.CharacterAggregate, error) {
	if value, ok := t.pending[id]; ok {
		return clonePostgresAggregate(value), nil
	}
	return loadCharacterTx(t.ctx, t.tx, id, true)
}

func (t *tradeTransaction) SaveCharacter(value persistence.CharacterAggregate) error {
	if err := persistence.ValidateAggregate(value); err != nil {
		return err
	}
	if value.Character.ID != t.session.PlayerAID && value.Character.ID != t.session.PlayerBID {
		return persistence.ErrInvalidAggregate
	}
	t.pending[value.Character.ID] = clonePostgresAggregate(value)
	if len(t.pending) == 2 {
		return t.flushCharacters()
	}
	return nil
}

func (t *tradeTransaction) flushCharacters() error {
	ids := []string{t.session.PlayerAID, t.session.PlayerBID}
	for index, id := range ids {
		value, ok := t.pending[id]
		if !ok {
			return persistence.ErrInvalidAggregate
		}
		var revision int64
		err := t.tx.QueryRow(t.ctx, `UPDATE characters SET revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$2 RETURNING revision`, id, value.Character.Revision).Scan(&revision)
		if errors.Is(err, pgx.ErrNoRows) {
			return persistence.ErrStaleRevision
		}
		if err != nil {
			return classifyError(err)
		}
		value.Character.Revision = revision
		t.pending[id] = value
		if index == 0 && t.store.tradeFailureInjector != nil {
			if err = t.store.tradeFailureInjector(trade.FailureAfterFirstCharacterWrite); err != nil {
				return err
			}
		}
	}
	for _, id := range ids {
		if _, err := t.tx.Exec(t.ctx, "DELETE FROM character_equipment WHERE character_id=$1", id); err != nil {
			return classifyError(err)
		}
	}
	for _, id := range ids {
		if _, err := t.tx.Exec(t.ctx, "DELETE FROM character_inventory_items WHERE character_id=$1", id); err != nil {
			return classifyError(err)
		}
	}
	for _, id := range ids {
		value := t.pending[id]
		for _, item := range value.Items {
			var slot any
			if item.EquipmentSlot != "" {
				slot = item.EquipmentSlot
			}
			if _, err := t.tx.Exec(t.ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,equipment_slot) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, item.InstanceID, id, item.DefinitionID, item.LegacyID, item.Name, item.ItemType, item.Quantity, item.SlotIndex, item.Location, slot); err != nil {
				return classifyError(err)
			}
		}
		for _, equipped := range value.Equipment {
			if _, err := t.tx.Exec(t.ctx, "INSERT INTO character_equipment(character_id,slot,item_instance_id) VALUES($1,$2,$3)", id, equipped.Slot, equipped.ItemInstanceID); err != nil {
				return classifyError(err)
			}
		}
	}
	return nil
}

func (t *tradeTransaction) LockItem(lock trade.ItemLock) error {
	_, err := t.tx.Exec(t.ctx, `INSERT INTO trade_item_locks(item_instance_id,trade_id,owner_character_id,quantity,created_at) VALUES($1,$2,$3,$4,$5)
        ON CONFLICT(item_instance_id) DO UPDATE SET quantity=excluded.quantity
        WHERE trade_item_locks.trade_id=excluded.trade_id AND trade_item_locks.owner_character_id=excluded.owner_character_id`, lock.ItemInstanceID, lock.TradeID, lock.OwnerCharacterID, lock.Quantity, lock.CreatedAt)
	if err != nil {
		return classifyTradeError(err)
	}
	var tradeID, owner string
	var quantity int
	if err = t.tx.QueryRow(t.ctx, "SELECT trade_id,owner_character_id,quantity FROM trade_item_locks WHERE item_instance_id=$1", lock.ItemInstanceID).Scan(&tradeID, &owner, &quantity); err != nil {
		return classifyTradeError(err)
	}
	if tradeID != lock.TradeID || owner != lock.OwnerCharacterID || quantity != lock.Quantity {
		return trade.ErrItemLocked
	}
	return nil
}

func (t *tradeTransaction) ValidateItemLock(want trade.ItemLock) error {
	var tradeID, owner string
	var quantity int
	err := t.tx.QueryRow(t.ctx, "SELECT trade_id,owner_character_id,quantity FROM trade_item_locks WHERE item_instance_id=$1", want.ItemInstanceID).Scan(&tradeID, &owner, &quantity)
	if errors.Is(err, pgx.ErrNoRows) || tradeID != want.TradeID || owner != want.OwnerCharacterID || quantity != want.Quantity {
		return trade.ErrItemLocked
	}
	return classifyTradeError(err)
}

func (t *tradeTransaction) UnlockItem(id string) error {
	_, err := t.tx.Exec(t.ctx, "DELETE FROM trade_item_locks WHERE item_instance_id=$1 AND trade_id=$2", id, t.session.TradeID)
	return classifyTradeError(err)
}
func (t *tradeTransaction) UnlockAll() error {
	_, err := t.tx.Exec(t.ctx, "DELETE FROM trade_item_locks WHERE trade_id=$1", t.session.TradeID)
	return classifyTradeError(err)
}
func (t *tradeTransaction) AppendAudit(event trade.AuditEvent) error {
	return insertAudit(t.ctx, t.tx, event)
}

func (s *Store) CreateTrade(ctx context.Context, value trade.Session, event trade.AuditEvent) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return classifyTradeError(err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO trade_sessions(trade_id,player_a_id,player_b_id,state,revision,player_a_confirmed_revision,player_b_confirmed_revision,created_at,updated_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, value.TradeID, value.PlayerAID, value.PlayerBID, value.State, value.Revision, value.PlayerAConfirmedRevision, value.PlayerBConfirmedRevision, value.CreatedAt, value.UpdatedAt, value.ExpiresAt)
	if err != nil {
		return classifyTradeError(err)
	}
	if err = insertAudit(ctx, tx, event); err != nil {
		return err
	}
	return classifyTradeError(tx.Commit(ctx))
}

func (s *Store) LoadTrade(ctx context.Context, id string) (trade.Session, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	defer tx.Rollback(ctx)
	value, err := loadTradeTx(ctx, tx, id, false)
	if err != nil {
		return trade.Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	return value, nil
}

func (s *Store) Transact(ctx context.Context, id string, fn func(trade.Transaction) error) (trade.Session, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	defer tx.Rollback(ctx)
	value, err := loadTradeTx(ctx, tx, id, true)
	if err != nil {
		return trade.Session{}, err
	}
	uow := &tradeTransaction{ctx: ctx, tx: tx, store: s, session: &value, pending: map[string]persistence.CharacterAggregate{}}
	if err = fn(uow); err != nil {
		return trade.Session{}, err
	}
	if len(uow.pending) != 0 && len(uow.pending) != 2 {
		return trade.Session{}, persistence.ErrInvalidAggregate
	}
	if err = persistTradeTx(ctx, tx, value); err != nil {
		return trade.Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	return value, nil
}

func (s *Store) IsItemLocked(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM trade_item_locks WHERE item_instance_id=$1)", id).Scan(&exists)
	return exists, classifyTradeError(err)
}

func (s *Store) AuditEvents(ctx context.Context, id string) ([]trade.AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `SELECT sequence,trade_id,kind,revision,previous_state,new_state,player_a_id,player_b_id,occurred_at,outcome FROM trade_audit_events WHERE trade_id=$1 ORDER BY sequence`, id)
	if err != nil {
		return nil, classifyTradeError(err)
	}
	defer rows.Close()
	var events []trade.AuditEvent
	for rows.Next() {
		var event trade.AuditEvent
		var a, b string
		if err = rows.Scan(&event.Sequence, &event.TradeID, &event.Kind, &event.Revision, &event.PreviousState, &event.NewState, &a, &b, &event.OccurredAt, &event.Outcome); err != nil {
			return nil, classifyTradeError(err)
		}
		event.ParticipantIDs = []string{a, b}
		events = append(events, event)
	}
	return events, classifyTradeError(rows.Err())
}

func (s *Store) RecordSettlementFailure(ctx context.Context, id string, event trade.AuditEvent) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return classifyTradeError(err)
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM trade_sessions WHERE trade_id=$1)", id).Scan(&exists); err != nil {
		return classifyTradeError(err)
	}
	if !exists {
		return trade.ErrNotFound
	}
	if err = insertAudit(ctx, tx, event); err != nil {
		return err
	}
	return classifyTradeError(tx.Commit(ctx))
}

func loadTradeTx(ctx context.Context, tx pgx.Tx, id string, lock bool) (trade.Session, error) {
	query := `SELECT trade_id,player_a_id,player_b_id,state,revision,player_a_confirmed_revision,player_b_confirmed_revision,created_at,updated_at,expires_at,completed_at,cancelled_at FROM trade_sessions WHERE trade_id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	var value trade.Session
	err := tx.QueryRow(ctx, query, id).Scan(&value.TradeID, &value.PlayerAID, &value.PlayerBID, &value.State, &value.Revision, &value.PlayerAConfirmedRevision, &value.PlayerBConfirmedRevision, &value.CreatedAt, &value.UpdatedAt, &value.ExpiresAt, &value.CompletedAt, &value.CancelledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return trade.Session{}, trade.ErrNotFound
	}
	if err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	rows, err := tx.Query(ctx, "SELECT player_id,item_instance_id,quantity FROM trade_offers WHERE trade_id=$1 ORDER BY player_id,item_instance_id", id)
	if err != nil {
		return trade.Session{}, classifyTradeError(err)
	}
	for rows.Next() {
		var player string
		var offer trade.OfferItem
		if err = rows.Scan(&player, &offer.ItemInstanceID, &offer.Quantity); err != nil {
			rows.Close()
			return trade.Session{}, classifyTradeError(err)
		}
		if player == value.PlayerAID {
			value.PlayerAOffer = append(value.PlayerAOffer, offer)
		} else if player == value.PlayerBID {
			value.PlayerBOffer = append(value.PlayerBOffer, offer)
		} else {
			rows.Close()
			return trade.Session{}, persistence.ErrInvalidAggregate
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return trade.Session{}, classifyTradeError(err)
	}
	rows.Close()
	var result trade.SettlementResult
	err = tx.QueryRow(ctx, "SELECT trade_id,settlement_id,completed_at FROM trade_settlements WHERE trade_id=$1", id).Scan(&result.TradeID, &result.SettlementID, &result.CompletedAt)
	if err == nil {
		value.Settlement = &result
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return trade.Session{}, classifyTradeError(err)
	}
	return value, nil
}

func persistTradeTx(ctx context.Context, tx pgx.Tx, value trade.Session) error {
	_, err := tx.Exec(ctx, `UPDATE trade_sessions SET state=$1,revision=$2,player_a_confirmed_revision=$3,player_b_confirmed_revision=$4,updated_at=$5,expires_at=$6,completed_at=$7,cancelled_at=$8 WHERE trade_id=$9`, value.State, value.Revision, value.PlayerAConfirmedRevision, value.PlayerBConfirmedRevision, value.UpdatedAt, value.ExpiresAt, value.CompletedAt, value.CancelledAt, value.TradeID)
	if err != nil {
		return classifyTradeError(err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM trade_offers WHERE trade_id=$1", value.TradeID); err != nil {
		return classifyTradeError(err)
	}
	for _, pair := range []struct {
		player string
		offers []trade.OfferItem
	}{{value.PlayerAID, value.PlayerAOffer}, {value.PlayerBID, value.PlayerBOffer}} {
		for _, offer := range pair.offers {
			if _, err = tx.Exec(ctx, "INSERT INTO trade_offers(trade_id,player_id,item_instance_id,quantity) VALUES($1,$2,$3,$4)", value.TradeID, pair.player, offer.ItemInstanceID, offer.Quantity); err != nil {
				return classifyTradeError(err)
			}
		}
	}
	if value.Settlement != nil {
		_, err = tx.Exec(ctx, `INSERT INTO trade_settlements(settlement_id,trade_id,completed_at) VALUES($1,$2,$3) ON CONFLICT(trade_id) DO NOTHING`, value.Settlement.SettlementID, value.TradeID, value.Settlement.CompletedAt)
		if err != nil {
			return classifyTradeError(err)
		}
	}
	return nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, event trade.AuditEvent) error {
	if len(event.ParticipantIDs) != 2 {
		return persistence.ErrInvalidAggregate
	}
	_, err := tx.Exec(ctx, `INSERT INTO trade_audit_events(trade_id,kind,revision,previous_state,new_state,player_a_id,player_b_id,occurred_at,outcome) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, event.TradeID, event.Kind, event.Revision, event.PreviousState, event.NewState, event.ParticipantIDs[0], event.ParticipantIDs[1], event.OccurredAt, event.Outcome)
	return classifyTradeError(err)
}

func loadCharacterTx(ctx context.Context, tx pgx.Tx, id string, lock bool) (persistence.CharacterAggregate, error) {
	var value persistence.CharacterAggregate
	c := &value.Character
	query := `SELECT id,account_id,name,class_id,class_name,level,exp,current_hp,current_mp,revision FROM characters WHERE id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	err := tx.QueryRow(ctx, query, id).Scan(&c.ID, &c.AccountID, &c.Name, &c.ClassID, &c.ClassName, &c.Level, &c.EXP, &c.HP, &c.MP, &c.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, persistence.ErrNotFound
	}
	if err != nil {
		return value, classifyTradeError(err)
	}
	if err = tx.QueryRow(ctx, "SELECT map_id,x,y FROM character_world_state WHERE character_id=$1", id).Scan(&value.World.MapID, &value.World.X, &value.World.Y); err != nil {
		return value, classifyTradeError(err)
	}
	rows, err := tx.Query(ctx, `SELECT instance_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,coalesce(equipment_slot,'') FROM character_inventory_items WHERE character_id=$1 ORDER BY slot_index,instance_id`, id)
	if err != nil {
		return value, classifyTradeError(err)
	}
	for rows.Next() {
		var item persistence.ItemInstance
		if err = rows.Scan(&item.InstanceID, &item.DefinitionID, &item.LegacyID, &item.Name, &item.ItemType, &item.Quantity, &item.SlotIndex, &item.Location, &item.EquipmentSlot); err != nil {
			rows.Close()
			return value, classifyTradeError(err)
		}
		value.Items = append(value.Items, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return value, classifyTradeError(err)
	}
	rows.Close()
	rows, err = tx.Query(ctx, "SELECT slot,item_instance_id FROM character_equipment WHERE character_id=$1 ORDER BY slot", id)
	if err != nil {
		return value, classifyTradeError(err)
	}
	for rows.Next() {
		var e persistence.EquippedItem
		if err = rows.Scan(&e.Slot, &e.ItemInstanceID); err != nil {
			rows.Close()
			return value, classifyTradeError(err)
		}
		value.Equipment = append(value.Equipment, e)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return value, classifyTradeError(err)
	}
	rows.Close()
	rows, err = tx.Query(ctx, "SELECT skill_id,learned,skill_level FROM character_skills WHERE character_id=$1 ORDER BY skill_id", id)
	if err != nil {
		return value, classifyTradeError(err)
	}
	for rows.Next() {
		var skill persistence.SkillState
		if err = rows.Scan(&skill.SkillID, &skill.Learned, &skill.Level); err != nil {
			rows.Close()
			return value, classifyTradeError(err)
		}
		value.Skills = append(value.Skills, skill)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return value, classifyTradeError(err)
	}
	rows.Close()
	return value, persistence.ValidateAggregate(value)
}

func clonePostgresAggregate(value persistence.CharacterAggregate) persistence.CharacterAggregate {
	value.Items = append([]persistence.ItemInstance(nil), value.Items...)
	value.Equipment = append([]persistence.EquippedItem(nil), value.Equipment...)
	value.Skills = append([]persistence.SkillState(nil), value.Skills...)
	return value
}

func classifyTradeError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && pgErr.ConstraintName == "trade_item_locks_pkey" {
			return trade.ErrItemLocked
		}
		if pgErr.Code == "23505" {
			return persistence.ErrConflict
		}
	}
	return err
}

var _ trade.Repository = (*Store)(nil)
