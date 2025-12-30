package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/garrettladley/thoop/internal/client/whoop"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	go_json "github.com/goccy/go-json"
)

type recoveryRepo struct {
	q sqlitec.Querier
}

func (r *recoveryRepo) Upsert(ctx context.Context, recovery *whoop.Recovery) error {
	var scoreJSON *string
	if recovery.Score != nil {
		data, err := go_json.Marshal(recovery.Score)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		s := string(data)
		scoreJSON = &s
	}

	err := r.q.UpsertRecovery(ctx, sqlitec.UpsertRecoveryParams{
		CycleID:    recovery.CycleID,
		SleepID:    recovery.SleepID,
		UserID:     recovery.UserID,
		CreatedAt:  recovery.CreatedAt,
		UpdatedAt:  recovery.UpdatedAt,
		ScoreState: string(recovery.ScoreState),
		ScoreJson:  scoreJSON,
	})
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *recoveryRepo) UpsertBatch(ctx context.Context, recoveries []whoop.Recovery) error {
	for i := range recoveries {
		if err := r.Upsert(ctx, &recoveries[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *recoveryRepo) Get(ctx context.Context, cycleID int64) (*whoop.Recovery, error) {
	row, err := r.q.GetRecovery(ctx, cycleID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return r.toDomain(row)
}

func (r *recoveryRepo) GetByCycleIDs(ctx context.Context, cycleIDs []int64) ([]whoop.Recovery, error) {
	if len(cycleIDs) == 0 {
		return []whoop.Recovery{}, nil
	}

	rows, err := r.q.GetRecoveriesByCycleIDs(ctx, cycleIDs)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return r.toDomainSlice(rows)
}

func (r *recoveryRepo) toDomain(row sqlitec.Recovery) (*whoop.Recovery, error) {
	recovery := &whoop.Recovery{
		CycleID:    row.CycleID,
		SleepID:    row.SleepID,
		UserID:     row.UserID,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		ScoreState: whoop.ScoreState(row.ScoreState),
	}

	if row.ScoreJson != nil {
		var score whoop.RecoveryScore
		if err := go_json.Unmarshal([]byte(*row.ScoreJson), &score); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		recovery.Score = &score
	}

	return recovery, nil
}

func (r *recoveryRepo) toDomainSlice(rows []sqlitec.Recovery) ([]whoop.Recovery, error) {
	recoveries := make([]whoop.Recovery, 0, len(rows))
	for _, row := range rows {
		recovery, err := r.toDomain(row)
		if err != nil {
			return nil, err
		}
		recoveries = append(recoveries, *recovery)
	}
	return recoveries, nil
}

func (r *recoveryRepo) Delete(ctx context.Context, cycleID int64) error {
	err := r.q.DeleteRecovery(ctx, cycleID)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
