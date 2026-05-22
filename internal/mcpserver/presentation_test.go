package mcpserver

import (
	"strings"
	"testing"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "zero", in: 0, want: "0m"},
		{name: "minutes", in: 42 * time.Minute, want: "42m"},
		{name: "hours", in: 2 * time.Hour, want: "2h"},
		{name: "hours and minutes", in: 7*time.Hour + 4*time.Minute, want: "7h4m"},
		{name: "rounds to minute", in: 89 * time.Second, want: "1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatDuration(tt.in); got != tt.want {
				t.Fatalf("formatDuration(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   float64
		want string
	}{
		{name: "integer", in: 92, want: "92%"},
		{name: "decimal", in: 91.6, want: "91.6%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatPercent(tt.in); got != tt.want {
				t.Fatalf("formatPercent(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMockSleepDetailSerializesReadableUnits(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 5, 22, 1, 18, 0, 0, fixedEastern())
	end := time.Date(2026, 5, 22, 8, 2, 0, 0, fixedEastern())
	sleep := &whoop.Sleep{
		ID:             "sleep-1",
		CycleID:        93845,
		UserID:         10129,
		CreatedAt:      start,
		UpdatedAt:      end,
		Start:          start,
		End:            end,
		TimezoneOffset: "-04:00",
		Nap:            false,
		ScoreState:     whoop.ScoreStateScored,
		Score: &whoop.SleepScore{
			StageSummary: whoop.SleepStages{
				TotalInBedTimeMilli:         25_440_000,
				TotalAwakeTimeMilli:         1_800_000,
				TotalLightSleepTimeMilli:    11_400_000,
				TotalSlowWaveSleepTimeMilli: 5_400_000,
				TotalREMSleepTimeMilli:      6_840_000,
				SleepCycleCount:             5,
				DisturbanceCount:            8,
			},
			SleepNeeded: whoop.SleepNeeded{
				BaselineMilli:             28_800_000,
				NeedFromSleepDebtMilli:    1_200_000,
				NeedFromRecentStrainMilli: 600_000,
			},
			RespiratoryRate:            16.1,
			SleepPerformancePercentage: 92,
			SleepConsistencyPercentage: 88,
			SleepEfficiencyPercentage:  91.6,
		},
	}

	out := singleEnvelope(mapSleepDetail(sleep), envelopeOptions{Source: testEnvelopeSource})

	assertContainsAll(t, out,
		"duration: 6h44m",
		"start_local_date: \"2026-05-22\"",
		"start_local_time: \"01:18\"",
		"total_in_bed_time_milli: 25440000",
		"total_in_bed_time: 7h4m",
		"baseline_milli: 28800000",
		"baseline: 8h",
		"respiratory_rate_unit: breaths_per_min",
		"sleep_performance_pct: 92%",
		"sleep_efficiency_pct: 91.6%",
	)
}

func TestMockSummariesSerializeReadableUnits(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 5, 22, 17, 12, 0, 0, fixedEastern())
	end := time.Date(2026, 5, 22, 17, 58, 0, 0, fixedEastern())
	distance := 1772.77
	altitudeGain := 46.64
	altitudeChange := -0.78
	items := mapWorkouts([]whoop.Workout{{
		ID:             "workout-1",
		SportName:      "running",
		Start:          start,
		End:            end,
		TimezoneOffset: "-04:00",
		ScoreState:     whoop.ScoreStateScored,
		Score: &whoop.WorkoutScore{
			Strain:              8.2,
			AverageHeartRate:    123,
			MaxHeartRate:        146,
			Kilojoule:           1569.34,
			DistanceMeter:       &distance,
			AltitudeGainMeter:   &altitudeGain,
			AltitudeChangeMeter: &altitudeChange,
			ZoneDurations: whoop.WorkoutZones{
				ZoneZeroMilli: 300_000,
				ZoneOneMilli:  600_000,
			},
		},
	}})

	out := textEnvelope(items, PageInput{MaxTokens: 1000}, envelopeOptions{Source: testEnvelopeSource})

	assertContainsAll(t, out,
		"duration: 46m",
		"average_heart_rate_unit: bpm",
		"max_heart_rate_unit: bpm",
		"kilojoule: 1569.34",
		"kilocalorie: 375.08",
		"distance_meter: 1772.77",
		"distance: 1772.77 m",
		"distance_kilometer: 1.77",
		"distance_mile: 1.1",
		"altitude_gain: 46.64 m",
		"altitude_gain_feet: 153.02",
		"zone_zero_milli: 300000",
		"zone_zero: 5m",
	)
}

func TestMockRecoverySummarySerializesReadableUnits(t *testing.T) {
	t.Parallel()

	recoveries := mapRecoveries([]whoop.Recovery{{
		CycleID:    93845,
		SleepID:    "sleep-1",
		UpdatedAt:  time.Date(2026, 5, 22, 11, 4, 0, 0, time.UTC),
		ScoreState: whoop.ScoreStateScored,
		Score: &whoop.RecoveryScore{
			RecoveryScore:    72,
			RestingHeartRate: 54,
			HRVRmssdMilli:    67.4,
			SpO2Percentage:   95.7,
			SkinTempCelsius:  33.7,
		},
	}})

	out := textEnvelope(recoveries, PageInput{MaxTokens: 1000}, envelopeOptions{Source: testEnvelopeSource})

	assertContainsAll(t, out,
		"recovery_score_pct: 72%",
		"resting_heart_rate_unit: bpm",
		"hrv_rmssd: 67.4 ms",
		"spo2_pct: 95.7%",
		"skin_temp_celsius: 33.7",
		"skin_temp_fahrenheit: 92.66",
	)
}

func fixedEastern() *time.Location {
	return time.FixedZone("EDT", -4*60*60)
}

func assertContainsAll(t *testing.T, out string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
