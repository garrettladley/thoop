package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test the WHOOP API client",
		Long:  "Fetches your profile and recent data to verify the client works.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := config.Read()
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			dbPath, err := paths.DB()
			if err != nil {
				return err
			}

			sqlDB, querier, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = sqlDB.Close() }()

			oauthCfg := oauth.NewConfig(cfg.Whoop)
			tokenSource := oauth.NewDBTokenSource(oauthCfg, querier)

			client := whoop.New(tokenSource, whoop.WithBaseURL(cfg.ProxyURL+"/api/whoop"))

			var failures int

			fmt.Println("\n[User.GetProfile]")
			profile, err := client.User.GetProfile(ctx)
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: %s %s (%s)\n", profile.FirstName, profile.LastName, profile.Email)
			}

			fmt.Println("\n[User.GetBodyMeasurement]")
			body, err := client.User.GetBodyMeasurement(ctx)
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: Height=%.2fm, Weight=%.1fkg, MaxHR=%d\n", body.HeightMeter, body.WeightKilogram, body.MaxHeartRate)
			}

			fmt.Println("\n[Cycle.List]")
			cycles, err := client.Cycle.List(ctx, &whoop.ListParams{Limit: 3})
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: %d cycles\n", len(cycles.Records))
				for _, c := range cycles.Records {
					fmt.Printf("    - id=%d, start=%s, score_state=%s\n", c.ID, c.Start.Format("2006-01-02"), c.ScoreState)
				}
			}

			if cycles != nil && len(cycles.Records) > 0 {
				cycleID := cycles.Records[0].ID

				fmt.Printf("\n[Cycle.Get] id=%d\n", cycleID)
				cycle, err := client.Cycle.Get(ctx, cycleID)
				if err != nil {
					fmt.Printf("  ERROR: %v\n", err)
					failures++
				} else {
					fmt.Printf("  OK: id=%d, score_state=%s\n", cycle.ID, cycle.ScoreState)
				}

				fmt.Printf("\n[Cycle.GetRecovery] cycleID=%d\n", cycleID)
				recovery, err := client.Cycle.GetRecovery(ctx, cycleID)
				if err != nil {
					fmt.Printf("  ERROR: %v\n", err)
					failures++
				} else {
					fmt.Printf("  OK: score_state=%s\n", recovery.ScoreState)
					if recovery.Score != nil {
						fmt.Printf("    recovery=%.0f%%, hrv=%.1f, rhr=%.0f\n",
							recovery.Score.RecoveryScore, recovery.Score.HRVRmssdMilli, recovery.Score.RestingHeartRate)
					}
				}

				fmt.Printf("\n[Cycle.GetSleep] cycleID=%d\n", cycleID)
				cycleSleep, err := client.Cycle.GetSleep(ctx, cycleID)
				if err != nil {
					fmt.Printf("  ERROR: %v\n", err)
					failures++
				} else {
					fmt.Printf("  OK: id=%s, nap=%v, score_state=%s\n", cycleSleep.ID, cycleSleep.Nap, cycleSleep.ScoreState)
				}
			}

			fmt.Println("\n[Recovery.List]")
			recoveries, err := client.Recovery.List(ctx, &whoop.ListParams{Limit: 3})
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: %d recoveries\n", len(recoveries.Records))
				for _, r := range recoveries.Records {
					if r.Score != nil {
						fmt.Printf("    - cycle=%d, recovery=%.0f%%, hrv=%.1f\n",
							r.CycleID, r.Score.RecoveryScore, r.Score.HRVRmssdMilli)
					}
				}
			}

			fmt.Println("\n[Sleep.List]")
			sleeps, err := client.Sleep.List(ctx, &whoop.ListParams{Limit: 3})
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: %d sleeps\n", len(sleeps.Records))
				for _, s := range sleeps.Records {
					fmt.Printf("    - id=%s, nap=%v, score_state=%s\n", s.ID, s.Nap, s.ScoreState)
				}
			}

			if sleeps != nil && len(sleeps.Records) > 0 {
				sleepID := sleeps.Records[0].ID

				fmt.Printf("\n[Sleep.Get] id=%s\n", sleepID)
				sleep, err := client.Sleep.Get(ctx, sleepID)
				if err != nil {
					fmt.Printf("  ERROR: %v\n", err)
					failures++
				} else {
					fmt.Printf("  OK: id=%s, nap=%v, score_state=%s\n", sleep.ID, sleep.Nap, sleep.ScoreState)
				}
			}

			fmt.Println("\n[Workout.List]")
			workouts, err := client.Workout.List(ctx, &whoop.ListParams{Limit: 3})
			if err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				failures++
			} else {
				fmt.Printf("  OK: %d workouts\n", len(workouts.Records))
				for _, w := range workouts.Records {
					fmt.Printf("    - id=%s, sport=%s, score_state=%s\n", w.ID, w.SportName, w.ScoreState)
				}
			}

			if workouts != nil && len(workouts.Records) > 0 {
				fmt.Println("\n[Workout.Get]")
				var gotOne bool
				for _, w := range workouts.Records {
					fmt.Printf("  trying id=%s (%s)... ", w.ID, w.SportName)
					workout, err := client.Workout.Get(ctx, w.ID)
					if err != nil {
						fmt.Printf("FAIL: %v\n", err)
						continue
					}
					fmt.Println("OK")
					fmt.Printf("    sport=%s, score_state=%s\n", workout.SportName, workout.ScoreState)
					if workout.Score != nil {
						fmt.Printf("    strain=%.1f, avg_hr=%d, max_hr=%d, kj=%.1f\n",
							workout.Score.Strain, workout.Score.AverageHeartRate,
							workout.Score.MaxHeartRate, workout.Score.Kilojoule)
					}
					gotOne = true
					break
				}
				if !gotOne {
					fmt.Println("  All workouts returned 404 - this may be a WHOOP API issue")
					failures++
				}
			}

			fmt.Println("\n" + "==========")
			if failures == 0 {
				fmt.Println("All endpoints passed!")
			} else {
				fmt.Printf("%d endpoint(s) failed\n", failures)
			}

			return nil
		},
	}
}
