package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type ShardStatus struct {
	ShardID    string    `db:"shard_id"`
	ShardChar  string    `db:"shard_char"`
	Start      int64     `db:"start"`
	End        int64     `db:"end"`
	Generation int64     `db:"generation"`
	Status     string    `db:"status"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

const (
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusFailed     = "failed"
)

const DefaultSeedStart uint64 = 1000000000

const (
	ShardStatusInsertCreateQuery = `INSERT INTO shard_status (shard_id, shard_char, start, end, status, created_at, updated_at) VALUES %s`
	ShardStatusSelectQuery       = `SELECT shard_id, shard_char, start, end, status, generation, updated_at FROM shard_status WHERE shard_id = ? AND shard_char = ?`
	ShardStatusUpdateStatusQuery = `UPDATE shard_status SET end = ?, updated_at = ?, generation = generation + 1, status = ? WHERE shard_id = ? AND shard_char = ? AND generation = ? AND status = ?`
)

func ExplodeKeyRange(shardID string) (byte, byte, bool) {
	splits := strings.Split(shardID, "-")
	var noop byte

	if len(splits) < 2 {
		return noop, noop, false
	}

	return splits[0][0], splits[1][0], true
}

type ShardStatusRepo struct {
	db *sql.DB

	seedStart uint64 // TODO:Might run out of space
}

func NewShardStatusRepo(db *sql.DB, seedStart uint64) *ShardStatusRepo {
	return &ShardStatusRepo{db: db, seedStart: seedStart}
}

func (repo *ShardStatusRepo) GetLastState(ctx context.Context, shardID, shardChar string) (*ShardStatus, error) {
	rows, err := repo.db.QueryContext(ctx, ShardStatusSelectQuery, shardID, shardChar)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if ok := rows.Next(); !ok {
		return nil, sql.ErrNoRows
	}

	shardStatus := &ShardStatus{}

	err = rows.Scan(
		&shardStatus.ShardID,
		&shardStatus.ShardChar,
		&shardStatus.Start,
		&shardStatus.End,
		&shardStatus.Generation,
		&shardStatus.UpdatedAt,
	)

	return shardStatus, err
}

func (repo *ShardStatusRepo) UpdateState(ctx context.Context, status *ShardStatus) error {
	log.Println("updating shard generation info", status.ShardID, status.ShardChar)

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		ShardStatusUpdateStatusQuery,
		status.End,
		time.Now().UTC(),
		status.Status,
		status.ShardID,
		status.ShardChar,
		status.Generation,
		status.Status,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (repo *ShardStatusRepo) SeedShit(ctx context.Context, keyRanges []string, offset uint64) error {
	log.Println("seeding db")

	query := ShardStatusInsertCreateQuery
	values := [][]interface{}{}
	now := time.Now().UTC()

	for _, keyRange := range keyRanges {
		splits := strings.Split(fmt.Sprint("%v", keyRange), "-")
		start, end := rune(splits[0][0]), rune(splits[1][0])

		for ch := start; ch < end; ch++ {
			values = append(values, []interface{}{
				keyRange,
				string(ch),
				repo.seedStart,
				offset + repo.seedStart,
				StatusProcessing,
				now,
				now,
			})
		}
	}

	columns := len(values[0])
	placeholders := fmt.Sprintf("(%s),", strings.TrimSuffix(strings.Repeat("?,", columns), ","))
	batchedHolders := strings.TrimSuffix(strings.Repeat(placeholders, len(values)), ",")

	query = fmt.Sprintf(query, batchedHolders)
	log.Println("query", query)

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	flatValues := []interface{}{}
	for _, value := range values {
		flatValues = append(flatValues, value...)
	}

	_, err = tx.ExecContext(ctx, query, flatValues...)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()

	return err

}
