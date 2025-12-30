package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	go_json "github.com/goccy/go-json"
)

type sleepRepo struct {
	q sqlitec.Querier
}

func (r *sleepRepo) Upsert(ctx context.Context, sleep *whoop.Sleep) error {
	var scoreJSON *string
	if sleep.Score != nil {
		data, err := go_json.Marshal(sleep.Score)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		s := string(data)
		scoreJSON = &s
	}

	var nap int64
	if sleep.Nap {
		nap = 1
	}

	err := r.q.UpsertSleep(ctx, sqlitec.UpsertSleepParams{
		ID:             sleep.ID,
		CycleID:        sleep.CycleID,
		V1ID:           sleep.V1ID,
		UserID:         sleep.UserID,
		CreatedAt:      sleep.CreatedAt,
		UpdatedAt:      sleep.UpdatedAt,
		Start:          sleep.Start,
		End:            sleep.End,
		TimezoneOffset: sleep.TimezoneOffset,
		Nap:            nap,
		ScoreState:     string(sleep.ScoreState),
		ScoreJson:      scoreJSON,
	})
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *sleepRepo) UpsertBatch(ctx context.Context, sleeps []whoop.Sleep) error {
	for i := range sleeps {
		if err := r.Upsert(ctx, &sleeps[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *sleepRepo) Get(ctx context.Context, id string) (*whoop.Sleep, error) {
	row, err := r.q.GetSleep(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return r.toDomain(row)
}

func (r *sleepRepo) GetByCycleID(ctx context.Context, cycleID int64) (*whoop.Sleep, error) {
	row, err := r.q.GetSleepByCycleID(ctx, cycleID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return r.toDomain(row)
}

func (r *sleepRepo) GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Sleep], error) {
	limit := int64(DefaultPageSize)
	if cursor != nil && cursor.Limit > 0 {
		limit = int64(cursor.Limit)
	}

	fetchLimit := limit + 1

	var rows []sqlitec.Sleep
	var err error

	if cursor != nil && cursor.Cursor != nil {
		rows, err = r.q.GetSleepsByDateRangeCursor(ctx, sqlitec.GetSleepsByDateRangeCursorParams{
			RangeStart: start,
			RangeEnd:   end,
			Cursor:     *cursor.Cursor,
			Limit:      fetchLimit,
		})
	} else {
		rows, err = r.q.GetSleepsByDateRange(ctx, sqlitec.GetSleepsByDateRangeParams{
			RangeStart: start,
			RangeEnd:   end,
			Limit:      fetchLimit,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}

	sleeps, err := r.toDomainSlice(rows)
	if err != nil {
		return nil, err
	}

	result := &CursorResult[whoop.Sleep]{
		Records: sleeps,
	}

	if hasMore && len(sleeps) > 0 {
		lastStart := sleeps[len(sleeps)-1].Start
		result.NextCursor = &lastStart
	}

	return result, nil
}

func (r *sleepRepo) toDomain(row sqlitec.Sleep) (*whoop.Sleep, error) {
	sleep := &whoop.Sleep{
		ID:             row.ID,
		CycleID:        row.CycleID,
		V1ID:           row.V1ID,
		UserID:         row.UserID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Start:          row.Start,
		End:            row.End,
		TimezoneOffset: row.TimezoneOffset,
		Nap:            row.Nap == 1,
		ScoreState:     whoop.ScoreState(row.ScoreState),
	}

	if row.ScoreJson != nil {
		var score whoop.SleepScore
		if err := json.Unmarshal([]byte(*row.ScoreJson), &score); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		sleep.Score = &score
	}

	return sleep, nil
}

func (r *sleepRepo) toDomainSlice(rows []sqlitec.Sleep) ([]whoop.Sleep, error) {
	sleeps := make([]whoop.Sleep, 0, len(rows))
	for _, row := range rows {
		sleep, err := r.toDomain(row)
		if err != nil {
			return nil, err
		}
		sleeps = append(sleeps, *sleep)
	}
	return sleeps, nil
}

func (r *sleepRepo) Delete(ctx context.Context, id string) error {
	err := r.q.DeleteSleep(ctx, id)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
