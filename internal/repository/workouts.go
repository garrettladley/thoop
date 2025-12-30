package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	go_json "github.com/goccy/go-json"
)

type workoutRepo struct {
	q sqlitec.Querier
}

func (r *workoutRepo) Upsert(ctx context.Context, workout *whoop.Workout) error {
	var scoreJSON *string
	if workout.Score != nil {
		data, err := go_json.Marshal(workout.Score)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		s := string(data)
		scoreJSON = &s
	}

	err := r.q.UpsertWorkout(ctx, sqlitec.UpsertWorkoutParams{
		ID:             workout.ID,
		V1ID:           workout.V1ID,
		UserID:         workout.UserID,
		CreatedAt:      workout.CreatedAt,
		UpdatedAt:      workout.UpdatedAt,
		Start:          workout.Start,
		End:            workout.End,
		TimezoneOffset: workout.TimezoneOffset,
		SportName:      workout.SportName,
		ScoreState:     string(workout.ScoreState),
		ScoreJson:      scoreJSON,
	})
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (r *workoutRepo) UpsertBatch(ctx context.Context, workouts []whoop.Workout) error {
	for i := range workouts {
		if err := r.Upsert(ctx, &workouts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *workoutRepo) Get(ctx context.Context, id string) (*whoop.Workout, error) {
	row, err := r.q.GetWorkout(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return r.toDomain(row)
}

func (r *workoutRepo) GetByDateRange(ctx context.Context, start, end time.Time, cursor *CursorParams) (*CursorResult[whoop.Workout], error) {
	limit := int64(DefaultPageSize)
	if cursor != nil && cursor.Limit > 0 {
		limit = int64(cursor.Limit)
	}

	fetchLimit := limit + 1

	var rows []sqlitec.Workout
	var err error

	if cursor != nil && cursor.Cursor != nil {
		rows, err = r.q.GetWorkoutsByDateRangeCursor(ctx, sqlitec.GetWorkoutsByDateRangeCursorParams{
			RangeStart: start,
			RangeEnd:   end,
			Cursor:     *cursor.Cursor,
			Limit:      fetchLimit,
		})
	} else {
		rows, err = r.q.GetWorkoutsByDateRange(ctx, sqlitec.GetWorkoutsByDateRangeParams{
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

	workouts, err := r.toDomainSlice(rows)
	if err != nil {
		return nil, err
	}

	result := &CursorResult[whoop.Workout]{
		Records: workouts,
	}

	if hasMore && len(workouts) > 0 {
		lastStart := workouts[len(workouts)-1].Start
		result.NextCursor = &lastStart
	}

	return result, nil
}

func (r *workoutRepo) toDomain(row sqlitec.Workout) (*whoop.Workout, error) {
	workout := &whoop.Workout{
		ID:             row.ID,
		V1ID:           row.V1ID,
		UserID:         row.UserID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Start:          row.Start,
		End:            row.End,
		TimezoneOffset: row.TimezoneOffset,
		SportName:      row.SportName,
		ScoreState:     whoop.ScoreState(row.ScoreState),
	}

	if row.ScoreJson != nil {
		var score whoop.WorkoutScore
		if err := go_json.Unmarshal([]byte(*row.ScoreJson), &score); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		workout.Score = &score
	}

	return workout, nil
}

func (r *workoutRepo) toDomainSlice(rows []sqlitec.Workout) ([]whoop.Workout, error) {
	workouts := make([]whoop.Workout, 0, len(rows))
	for _, row := range rows {
		workout, err := r.toDomain(row)
		if err != nil {
			return nil, err
		}
		workouts = append(workouts, *workout)
	}
	return workouts, nil
}

func (r *workoutRepo) Delete(ctx context.Context, id string) error {
	err := r.q.DeleteWorkout(ctx, id)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
