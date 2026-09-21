package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"fractallegend/game-server/internal/persistence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	pool                  *pgxpool.Pool
	tradeFailureInjector  func(string) error
	ledgerFailureInjector func(string) error
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("%w: database URL is required", persistence.ErrUnavailable)
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid database configuration", persistence.ErrUnavailable)
	}
	config.MaxConns = 8
	config.MinConns = 0
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	if config.ConnConfig.ConnectTimeout == 0 {
		config.ConnConfig.ConnectTimeout = 5 * time.Second
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%w: connection pool creation failed", persistence.ErrUnavailable)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: database connection failed", persistence.ErrUnavailable)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Migrate(ctx context.Context) error { return s.MigrateFS(ctx, migrations) }

func (s *Store) MigrateFS(ctx context.Context, source fs.FS) error {
	if s == nil || s.pool == nil {
		return persistence.ErrUnavailable
	}
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
        version bigint PRIMARY KEY,
        name text NOT NULL,
        applied_at timestamptz NOT NULL DEFAULT now()
    )`); err != nil {
		return fmt.Errorf("migration registry: %w", err)
	}
	entries, err := fs.ReadDir(source, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return fmt.Errorf("invalid migration name %q", entry.Name())
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil || version < 1 {
			return fmt.Errorf("invalid migration version %q", entry.Name())
		}
		contents, err := fs.ReadFile(source, path.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %d: %w", version, err)
		}
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}
		var exists bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists)
		if err == nil && !exists {
			_, err = tx.Exec(ctx, string(contents))
		}
		if err == nil && !exists {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version,name) VALUES($1,$2)", version, entry.Name())
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %d failed: %w", version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}
	return nil
}

func (s *Store) CreateAccount(ctx context.Context, value persistence.Account) error {
	if value.ID == "" {
		return persistence.ErrInvalidAggregate
	}
	now := time.Now().UTC()
	if value.CreatedAt.IsZero() {
		value.CreatedAt = now
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}
	_, err := s.pool.Exec(ctx, "INSERT INTO accounts(id,created_at,updated_at) VALUES($1,$2,$3)", value.ID, value.CreatedAt, value.UpdatedAt)
	return classifyError(err)
}

func (s *Store) LoadAccount(ctx context.Context, id string) (persistence.Account, error) {
	var value persistence.Account
	err := s.pool.QueryRow(ctx, "SELECT id,created_at,updated_at FROM accounts WHERE id=$1", id).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return persistence.Account{}, persistence.ErrNotFound
	}
	return value, classifyError(err)
}

func (s *Store) CreateCharacter(ctx context.Context, value persistence.CharacterAggregate) (int64, error) {
	if err := persistence.ValidateAggregate(value); err != nil {
		return 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, classifyError(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO characters(id,account_id,name,class_id,class_name,level,exp,current_hp,current_mp,revision)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,1)`, value.Character.ID, value.Character.AccountID, value.Character.Name, value.Character.ClassID, value.Character.ClassName, value.Character.Level, value.Character.EXP, value.Character.HP, value.Character.MP); err != nil {
		return 0, classifyCreateError(err)
	}
	if err = replaceChildren(ctx, tx, value); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, classifyError(err)
	}
	return 1, nil
}

func (s *Store) SaveCharacter(ctx context.Context, value persistence.CharacterAggregate, expectedRevision int64) (int64, error) {
	if err := persistence.ValidateAggregate(value); err != nil {
		return 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, classifyError(err)
	}
	defer tx.Rollback(ctx)
	var revision int64
	err = tx.QueryRow(ctx, `UPDATE characters SET name=$1,class_id=$2,class_name=$3,level=$4,exp=$5,current_hp=$6,current_mp=$7,revision=revision+1,updated_at=now()
        WHERE id=$8 AND account_id=$9 AND revision=$10 RETURNING revision`, value.Character.Name, value.Character.ClassID, value.Character.ClassName, value.Character.Level, value.Character.EXP, value.Character.HP, value.Character.MP, value.Character.ID, value.Character.AccountID, expectedRevision).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if queryErr := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM characters WHERE id=$1)", value.Character.ID).Scan(&exists); queryErr != nil {
			return 0, classifyError(queryErr)
		}
		if exists {
			return 0, persistence.ErrStaleRevision
		}
		return 0, persistence.ErrNotFound
	}
	if err != nil {
		return 0, classifyError(err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM character_equipment WHERE character_id=$1", value.Character.ID); err != nil {
		return 0, classifyError(err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM character_inventory_items WHERE character_id=$1", value.Character.ID); err != nil {
		return 0, classifyError(err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM character_skills WHERE character_id=$1", value.Character.ID); err != nil {
		return 0, classifyError(err)
	}
	if err = replaceChildren(ctx, tx, value); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, classifyError(err)
	}
	return revision, nil
}

func replaceChildren(ctx context.Context, tx pgx.Tx, value persistence.CharacterAggregate) error {
	if _, err := tx.Exec(ctx, `INSERT INTO character_world_state(character_id,map_id,x,y) VALUES($1,$2,$3,$4)
        ON CONFLICT(character_id) DO UPDATE SET map_id=excluded.map_id,x=excluded.x,y=excluded.y`, value.Character.ID, value.World.MapID, value.World.X, value.World.Y); err != nil {
		return classifyError(err)
	}
	for _, item := range value.Items {
		var slot any
		if item.EquipmentSlot != "" {
			slot = item.EquipmentSlot
		}
		if _, err := tx.Exec(ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,equipment_slot)
            VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, item.InstanceID, value.Character.ID, item.DefinitionID, item.LegacyID, item.Name, item.ItemType, item.Quantity, item.SlotIndex, item.Location, slot); err != nil {
			return classifyError(err)
		}
	}
	for _, equipped := range value.Equipment {
		if _, err := tx.Exec(ctx, "INSERT INTO character_equipment(character_id,slot,item_instance_id) VALUES($1,$2,$3)", value.Character.ID, equipped.Slot, equipped.ItemInstanceID); err != nil {
			return classifyError(err)
		}
	}
	for _, skill := range value.Skills {
		if _, err := tx.Exec(ctx, "INSERT INTO character_skills(character_id,skill_id,learned,skill_level) VALUES($1,$2,$3,$4)", value.Character.ID, skill.SkillID, skill.Learned, skill.Level); err != nil {
			return classifyError(err)
		}
	}
	return nil
}

func (s *Store) LoadCharacter(ctx context.Context, id string) (persistence.CharacterAggregate, error) {
	var value persistence.CharacterAggregate
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	defer tx.Rollback(ctx)
	c := &value.Character
	err = tx.QueryRow(ctx, `SELECT id,account_id,name,class_id,class_name,level,exp,current_hp,current_mp,revision FROM characters WHERE id=$1`, id).Scan(&c.ID, &c.AccountID, &c.Name, &c.ClassID, &c.ClassName, &c.Level, &c.EXP, &c.HP, &c.MP, &c.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return persistence.CharacterAggregate{}, persistence.ErrNotFound
	}
	if err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	if err = tx.QueryRow(ctx, "SELECT map_id,x,y FROM character_world_state WHERE character_id=$1", id).Scan(&value.World.MapID, &value.World.X, &value.World.Y); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return persistence.CharacterAggregate{}, persistence.ErrInvalidAggregate
		}
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	rows, err := tx.Query(ctx, `SELECT instance_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,coalesce(equipment_slot,'')
        FROM character_inventory_items WHERE character_id=$1 ORDER BY slot_index,instance_id`, id)
	if err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	for rows.Next() {
		var item persistence.ItemInstance
		if err = rows.Scan(&item.InstanceID, &item.DefinitionID, &item.LegacyID, &item.Name, &item.ItemType, &item.Quantity, &item.SlotIndex, &item.Location, &item.EquipmentSlot); err != nil {
			rows.Close()
			return persistence.CharacterAggregate{}, classifyError(err)
		}
		value.Items = append(value.Items, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	rows.Close()
	rows, err = tx.Query(ctx, "SELECT slot,item_instance_id FROM character_equipment WHERE character_id=$1 ORDER BY slot", id)
	if err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	for rows.Next() {
		var equipped persistence.EquippedItem
		if err = rows.Scan(&equipped.Slot, &equipped.ItemInstanceID); err != nil {
			rows.Close()
			return persistence.CharacterAggregate{}, classifyError(err)
		}
		value.Equipment = append(value.Equipment, equipped)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	rows.Close()
	rows, err = tx.Query(ctx, "SELECT skill_id,learned,skill_level FROM character_skills WHERE character_id=$1 ORDER BY skill_id", id)
	if err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	for rows.Next() {
		var skill persistence.SkillState
		if err = rows.Scan(&skill.SkillID, &skill.Learned, &skill.Level); err != nil {
			rows.Close()
			return persistence.CharacterAggregate{}, classifyError(err)
		}
		value.Skills = append(value.Skills, skill)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	rows.Close()
	if err = persistence.ValidateAggregate(value); err != nil {
		return persistence.CharacterAggregate{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return persistence.CharacterAggregate{}, classifyError(err)
	}
	return value, nil
}

func classifyCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return persistence.ErrAccountNotFound
		case "23505":
			return persistence.ErrConflict
		}
	}
	return classifyError(err)
}

func classifyError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return persistence.ErrConflict
	}
	return err
}
