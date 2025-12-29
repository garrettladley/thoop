package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/sqlc"
	go_json "github.com/goccy/go-json"
)

type recoveryRepo struct {
	q sqlc.Querier
}

func (r *recoveryRepo) Upsert(ctx context.Context, recovery *whoop.Recovery) error {
	var scoreJSON *string
	if recovery.Score != nil {
		data, err := go_json.Marshal(recovery.Score)
		if err != nil {
			return err
		}
		s := string(data)
		scoreJSON = &s
	}

	return r.q.UpsertRecovery(ctx, sqlc.UpsertRecoveryParams{
		CycleID:    recovery.CycleID,
		SleepID:    recovery.SleepID,
		UserID:     recovery.UserID,
		CreatedAt:  recovery.CreatedAt,
		UpdatedAt:  recovery.UpdatedAt,
		ScoreState: string(recovery.ScoreState),
		ScoreJson:  scoreJSON,
	})
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
		return nil, err
	}
	return r.toDomain(row)
}

func (r *recoveryRepo) GetByCycleIDs(ctx context.Context, cycleIDs []int64) ([]whoop.Recovery, error) {
	if len(cycleIDs) == 0 {
		return []whoop.Recovery{}, nil
	}

	rows, err := r.q.GetRecoveriesByCycleIDs(ctx, cycleIDs)
	if err != nil {
		return nil, err
	}
	return r.toDomainSlice(rows)
}

func (r *recoveryRepo) toDomain(row sqlc.Recovery) (*whoop.Recovery, error) {
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
			return nil, err
		}
		recovery.Score = &score
	}

	return recovery, nil
}

func (r *recoveryRepo) toDomainSlice(rows []sqlc.Recovery) ([]whoop.Recovery, error) {
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
	return r.q.DeleteRecovery(ctx, cycleID)
}
