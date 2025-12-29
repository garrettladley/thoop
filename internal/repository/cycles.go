package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	go_json "github.com/goccy/go-json"
)

type cycleRepo struct {
	q sqlitec.Querier
}

func (r *cycleRepo) Upsert(ctx context.Context, cycle *whoop.Cycle) error {
	var scoreJSON *string
	if cycle.Score != nil {
		data, err := go_json.Marshal(cycle.Score)
		if err != nil {
			return err
		}
		s := string(data)
		scoreJSON = &s
	}

	return r.q.UpsertCycle(ctx, sqlitec.UpsertCycleParams{
		ID:             cycle.ID,
		UserID:         cycle.UserID,
		CreatedAt:      cycle.CreatedAt,
		UpdatedAt:      cycle.UpdatedAt,
		Start:          cycle.Start,
		End:            cycle.End,
		TimezoneOffset: cycle.TimezoneOffset,
		ScoreState:     string(cycle.ScoreState),
		ScoreJson:      scoreJSON,
	})
}

func (r *cycleRepo) Get(ctx context.Context, id int64) (*whoop.Cycle, error) {
	row, err := r.q.GetCycle(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain(row)
}

func (r *cycleRepo) GetLatest(ctx context.Context, limit int) ([]whoop.Cycle, error) {
	rows, err := r.q.GetLatestCycles(ctx, int64(limit))
	if err != nil {
		return nil, err
	}
	return r.toDomainSlice(rows)
}

func (r *cycleRepo) UpsertBatch(ctx context.Context, cycles []whoop.Cycle) error {
	for i := range cycles {
		if err := r.Upsert(ctx, &cycles[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *cycleRepo) GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Cycle], error) {
	limit := int64(DefaultPageSize)
	if cursor != nil && cursor.Limit > 0 {
		limit = int64(cursor.Limit)
	}

	fetchLimit := limit + 1

	var rows []sqlitec.Cycle
	var err error

	if cursor != nil && cursor.Cursor != nil {
		rows, err = r.q.GetCyclesByDateRangeCursor(ctx, sqlitec.GetCyclesByDateRangeCursorParams{
			RangeStart: start,
			RangeEnd:   end,
			Cursor:     *cursor.Cursor,
			Limit:      fetchLimit,
		})
	} else {
		rows, err = r.q.GetCyclesByDateRange(ctx, sqlitec.GetCyclesByDateRangeParams{
			RangeStart: start,
			RangeEnd:   end,
			Limit:      fetchLimit,
		})
	}
	if err != nil {
		return nil, err
	}

	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}

	cycles, err := r.toDomainSlice(rows)
	if err != nil {
		return nil, err
	}

	result := &CursorResult[whoop.Cycle]{
		Records: cycles,
	}

	if hasMore && len(cycles) > 0 {
		lastStart := cycles[len(cycles)-1].Start
		result.NextCursor = &lastStart
	}

	return result, nil
}

func (r *cycleRepo) GetPending(ctx context.Context) ([]whoop.Cycle, error) {
	rows, err := r.q.GetPendingCycles(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainSlice(rows)
}

func (r *cycleRepo) toDomain(row sqlitec.Cycle) (*whoop.Cycle, error) {
	cycle := &whoop.Cycle{
		ID:             row.ID,
		UserID:         row.UserID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Start:          row.Start,
		End:            row.End,
		TimezoneOffset: row.TimezoneOffset,
		ScoreState:     whoop.ScoreState(row.ScoreState),
	}

	if row.ScoreJson != nil {
		var score whoop.CycleScore
		if err := go_json.Unmarshal([]byte(*row.ScoreJson), &score); err != nil {
			return nil, err
		}
		cycle.Score = &score
	}

	return cycle, nil
}

func (r *cycleRepo) toDomainSlice(rows []sqlitec.Cycle) ([]whoop.Cycle, error) {
	cycles := make([]whoop.Cycle, 0, len(rows))
	for _, row := range rows {
		cycle, err := r.toDomain(row)
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, *cycle)
	}
	return cycles, nil
}
