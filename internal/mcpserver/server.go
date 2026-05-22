package mcpserver

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/garrettladley/thoop"
	"github.com/garrettladley/thoop/internal/cache"
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/session"
	"github.com/garrettladley/thoop/internal/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const unitBPM = "bpm"

type Server struct {
	cache cache.CacheService
	repo  *repository.Repository
}

func RunStdio(ctx context.Context) error {
	if _, err := paths.EnsureDir(); err != nil {
		return fmt.Errorf("ensure thoop directory: %w", err)
	}

	dbPath, err := paths.DB()
	if err != nil {
		return fmt.Errorf("get database path: %w", err)
	}

	db, err := sqlite.New(ctx, sqlite.DefaultConfig(dbPath))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	kr := keyring.NewOSKeyring()
	if !kr.Available() {
		return fmt.Errorf("OS keyring is not available")
	}

	tokenSource := oauth.NewProxyTokenSource(config.ServerURL, db, kr)
	apiKey, _ := kr.Get(keyring.KeyAPIKey)
	sessionID := session.NewID()

	client := whoop.New(tokenSource,
		whoop.WithProxyURL(config.ServerURL+"/api/whoop"),
		whoop.WithSessionID(sessionID),
		whoop.WithAPIKey(apiKey),
	)
	repo := repository.New(db)
	cacheSvc := cache.NewService(client, repo)

	server := New(cacheSvc, repo)
	return server.Run(ctx)
}

func New(cacheSvc cache.CacheService, repo *repository.Repository) *Server {
	return &Server{cache: cacheSvc, repo: repo}
}

func (s *Server) Run(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "thoop", Version: thoop.Version}, nil)
	s.registerTools(server)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("run MCP server: %w", err)
	}
	return nil
}

func (s *Server) registerTools(server *mcp.Server) {
	readOnly := true
	openWorld := true

	tool := func(name string) *mcp.Tool {
		return &mcp.Tool{
			Name:        name,
			Description: ToolDescription(name),
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  readOnly,
				OpenWorldHint: &openWorld,
			},
		}
	}

	mcp.AddTool(server, tool("list_thoop_skills"), s.listSkills)
	mcp.AddTool(server, tool("load_thoop_skill"), s.loadSkill)
	mcp.AddTool(server, tool("get_latest_metrics"), s.getLatestMetrics)
	mcp.AddTool(server, tool("get_cycle"), s.getCycle)
	mcp.AddTool(server, tool("get_recovery"), s.getRecovery)
	mcp.AddTool(server, tool("get_sleep"), s.getSleep)
	mcp.AddTool(server, tool("list_cycles"), s.listCycles)
	mcp.AddTool(server, tool("list_recoveries"), s.listRecoveries)
	mcp.AddTool(server, tool("list_sleeps"), s.listSleeps)
	mcp.AddTool(server, tool("list_workouts"), s.listWorkouts)
}

type emptyInput struct{}

type loadSkillInput struct {
	SkillName    string `json:"skill_name" jsonschema:"Exact skill name, for example thoop/metrics."`
	HeaderOnly   bool   `json:"header_only,omitempty" jsonschema:"If true, return only summary, related skills, and resources."`
	ResourcePath string `json:"resource_path,omitempty" jsonschema:"Reserved for future bundled references."`
}

type dateInput struct {
	Date string `json:"date" jsonschema:"Date in YYYY-MM-DD format, interpreted in the local timezone."`
}

type cycleOrDateInput struct {
	Date    string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format, interpreted in the local timezone."`
	CycleID int64  `json:"cycle_id,omitempty" jsonschema:"WHOOP cycle ID. If provided, it takes precedence over date."`
}

type rangeInput struct {
	StartDate string `json:"start_date" jsonschema:"Inclusive start date in YYYY-MM-DD format."`
	EndDate   string `json:"end_date" jsonschema:"Inclusive end date in YYYY-MM-DD format."`
	PageInput
}

func (s *Server) listSkills(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	skills, err := ListSkills()
	if err != nil {
		return nil, nil, err
	}

	headers := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		headers = append(headers, map[string]any{
			"name":        skill.Name,
			"description": skill.Description,
			"related":     skill.Related,
		})
	}
	return textResult(textEnvelope(headers, PageInput{}, envelopeOptions{Source: "embedded_docs"})), nil, nil
}

func (s *Server) loadSkill(_ context.Context, _ *mcp.CallToolRequest, input loadSkillInput) (*mcp.CallToolResult, any, error) {
	if input.ResourcePath != "" {
		return nil, nil, fmt.Errorf("skill resources are not available yet")
	}
	skill, err := LoadSkill(input.SkillName)
	if err != nil {
		return nil, nil, err
	}

	if input.HeaderOnly {
		header := map[string]any{
			"name":        skill.Name,
			"description": skill.Description,
			"related":     skill.Related,
			"resources":   []string{},
		}
		return textResult(singleEnvelope(header, envelopeOptions{Source: "embedded_docs"})), nil, nil
	}

	return textResult(skill.Content), nil, nil
}

func (s *Server) getLatestMetrics(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	data, err := s.cache.GetHistoricalData(ctx, time.Now(), 30)
	if err != nil {
		return nil, nil, fmt.Errorf("get historical data: %w", err)
	}

	slices.SortFunc(data.Cycles, func(a, b whoop.Cycle) int { return b.Start.Compare(a.Start) })
	slices.SortFunc(data.Sleeps, func(a, b whoop.Sleep) int { return b.Start.Compare(a.Start) })
	slices.SortFunc(data.Workouts, func(a, b whoop.Workout) int { return b.Start.Compare(a.Start) })
	slices.SortFunc(data.Recoveries, func(a, b whoop.Recovery) int { return b.CreatedAt.Compare(a.CreatedAt) })

	summary := map[string]any{
		"cycle":    firstOrNil(mapCycles(data.Cycles)),
		"recovery": firstOrNil(mapRecoveries(data.Recoveries)),
		"sleep":    firstOrNil(mapSleeps(data.Sleeps)),
		"workout":  firstOrNil(mapWorkouts(data.Workouts)),
	}
	return textResult(singleEnvelope(summary, s.cacheEnvelopeOptions(data.FromCache, false))), nil, nil
}

func (s *Server) getCycle(ctx context.Context, _ *mcp.CallToolRequest, input dateInput) (*mcp.CallToolResult, any, error) {
	date, err := parseDate(input.Date)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetCycleForDate(ctx, date)
	if err != nil {
		return nil, nil, fmt.Errorf("get cycle for date: %w", err)
	}
	return textResult(singleEnvelope(result.Data, s.cacheEnvelopeOptions(result.FromCache, false))), nil, nil
}

func (s *Server) getRecovery(ctx context.Context, _ *mcp.CallToolRequest, input cycleOrDateInput) (*mcp.CallToolResult, any, error) {
	cycleID, err := s.resolveCycleID(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetRecoveryForCycle(ctx, cycleID)
	if err != nil {
		return nil, nil, fmt.Errorf("get recovery for cycle: %w", err)
	}
	return textResult(singleEnvelope(result.Data, s.cacheEnvelopeOptions(result.FromCache, false))), nil, nil
}

func (s *Server) getSleep(ctx context.Context, _ *mcp.CallToolRequest, input cycleOrDateInput) (*mcp.CallToolResult, any, error) {
	cycleID, err := s.resolveCycleID(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetSleepForCycle(ctx, cycleID)
	if err != nil {
		return nil, nil, fmt.Errorf("get sleep for cycle: %w", err)
	}
	return textResult(singleEnvelope(mapSleepDetail(result.Data), s.cacheEnvelopeOptions(result.FromCache, false))), nil, nil
}

func (s *Server) listCycles(ctx context.Context, _ *mcp.CallToolRequest, input rangeInput) (*mcp.CallToolResult, any, error) {
	start, end, err := parseRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetCyclesForRange(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("get cycles for range: %w", err)
	}
	opts := s.cacheEnvelopeOptions(result.FromCache, result.PartialCache).withNextCall("list_cycles", input.nextArgs())
	return textResult(textEnvelope(mapCycles(result.Records), input.PageInput, opts)), nil, nil
}

func (s *Server) listRecoveries(ctx context.Context, _ *mcp.CallToolRequest, input rangeInput) (*mcp.CallToolResult, any, error) {
	start, end, err := parseRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetRecoveriesForRange(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("get recoveries for range: %w", err)
	}
	opts := s.cacheEnvelopeOptions(result.FromCache, result.PartialCache).withNextCall("list_recoveries", input.nextArgs())
	return textResult(textEnvelope(mapRecoveries(result.Records), input.PageInput, opts)), nil, nil
}

func (s *Server) listSleeps(ctx context.Context, _ *mcp.CallToolRequest, input rangeInput) (*mcp.CallToolResult, any, error) {
	start, end, err := parseRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetSleepsForRange(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("get sleeps for range: %w", err)
	}
	opts := s.cacheEnvelopeOptions(result.FromCache, result.PartialCache).withNextCall("list_sleeps", input.nextArgs())
	return textResult(textEnvelope(mapSleeps(result.Records), input.PageInput, opts)), nil, nil
}

func (s *Server) listWorkouts(ctx context.Context, _ *mcp.CallToolRequest, input rangeInput) (*mcp.CallToolResult, any, error) {
	start, end, err := parseRange(input.StartDate, input.EndDate)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.cache.GetWorkoutsForDateRange(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("get workouts for date range: %w", err)
	}
	opts := s.cacheEnvelopeOptions(result.FromCache, false).withNextCall("list_workouts", input.nextArgs())
	return textResult(textEnvelope(mapWorkouts(result.Data), input.PageInput, opts)), nil, nil
}

func (s *Server) resolveCycleID(ctx context.Context, input cycleOrDateInput) (int64, error) {
	if input.CycleID != 0 {
		return input.CycleID, nil
	}
	if input.Date == "" {
		return 0, fmt.Errorf("provide either cycle_id or date")
	}
	date, err := parseDate(input.Date)
	if err != nil {
		return 0, err
	}
	result, err := s.cache.GetCycleForDate(ctx, date)
	if err != nil {
		return 0, fmt.Errorf("get cycle for date: %w", err)
	}
	if result.Data == nil {
		return 0, fmt.Errorf("no cycle found for %s", input.Date)
	}
	return result.Data.ID, nil
}

func (s *Server) cacheEnvelopeOptions(fromCache bool, partialCache bool) envelopeOptions {
	opts := envelopeOptions{
		Source:       "cache_service",
		FromCache:    &fromCache,
		PartialCache: partialCache,
	}
	if s.repo != nil {
		if state, err := s.repo.SyncState.Get(context.Background()); err == nil && state != nil {
			opts.LastSync = state.LastFullSync
		}
	}
	return opts
}

func (opts envelopeOptions) withNextCall(toolName string, args map[string]any) envelopeOptions {
	opts.ToolName = toolName
	opts.NextArgs = args
	return opts
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %q as YYYY-MM-DD: %w", value, err)
	}
	return parsed, nil
}

func parseRange(startValue, endValue string) (time.Time, time.Time, error) {
	start, err := parseDate(startValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseDate(endValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end_date must be on or after start_date")
	}
	return start, end.Add(24 * time.Hour), nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func firstOrNil[T any](items []T) any {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func (input dateInput) String() string {
	return input.Date
}

func (input cycleOrDateInput) String() string {
	if input.CycleID != 0 {
		return strconv.FormatInt(input.CycleID, 10)
	}
	return input.Date
}

func (input rangeInput) nextArgs() map[string]any {
	args := map[string]any{
		"start_date": input.StartDate,
		"end_date":   input.EndDate,
	}
	if input.MaxTokens > 0 {
		args["max_tokens"] = input.MaxTokens
	}
	return args
}

type cycleSummary struct {
	ID             int64      `json:"id" yaml:"id"`
	Start          time.Time  `json:"start" yaml:"start"`
	StartLocalDate string     `json:"start_local_date" yaml:"start_local_date"`
	StartLocalTime string     `json:"start_local_time" yaml:"start_local_time"`
	End            *time.Time `json:"end,omitempty" yaml:"end,omitempty"`
	EndLocalDate   *string    `json:"end_local_date,omitempty" yaml:"end_local_date,omitempty"`
	EndLocalTime   *string    `json:"end_local_time,omitempty" yaml:"end_local_time,omitempty"`
	Duration       *string    `json:"duration,omitempty" yaml:"duration,omitempty"`
	TimezoneOffset string     `json:"timezone_offset" yaml:"timezone_offset"`
	ScoreState     string     `json:"score_state" yaml:"score_state"`
	Strain         *float64   `json:"strain,omitempty" yaml:"strain,omitempty"`
	Kilojoule      *float64   `json:"kilojoule,omitempty" yaml:"kilojoule,omitempty"`
	Kilocalorie    *float64   `json:"kilocalorie,omitempty" yaml:"kilocalorie,omitempty"`
	AverageHR      *int       `json:"average_heart_rate,omitempty" yaml:"average_heart_rate,omitempty"`
	AverageHRUnit  string     `json:"average_heart_rate_unit,omitempty" yaml:"average_heart_rate_unit,omitempty"`
	MaxHR          *int       `json:"max_heart_rate,omitempty" yaml:"max_heart_rate,omitempty"`
	MaxHRUnit      string     `json:"max_heart_rate_unit,omitempty" yaml:"max_heart_rate_unit,omitempty"`
}

type recoverySummary struct {
	CycleID            int64     `json:"cycle_id" yaml:"cycle_id"`
	SleepID            string    `json:"sleep_id" yaml:"sleep_id"`
	UpdatedAt          time.Time `json:"updated_at" yaml:"updated_at"`
	ScoreState         string    `json:"score_state" yaml:"score_state"`
	RecoveryScore      *float64  `json:"recovery_score,omitempty" yaml:"recovery_score,omitempty"`
	RecoveryScorePct   *string   `json:"recovery_score_pct,omitempty" yaml:"recovery_score_pct,omitempty"`
	RestingHeartRate   *float64  `json:"resting_heart_rate,omitempty" yaml:"resting_heart_rate,omitempty"`
	RestingHRUnit      string    `json:"resting_heart_rate_unit,omitempty" yaml:"resting_heart_rate_unit,omitempty"`
	HRVRmssdMilli      *float64  `json:"hrv_rmssd_milli,omitempty" yaml:"hrv_rmssd_milli,omitempty"`
	HRVRmssd           *string   `json:"hrv_rmssd,omitempty" yaml:"hrv_rmssd,omitempty"`
	SpO2Percentage     *float64  `json:"spo2_percentage,omitempty" yaml:"spo2_percentage,omitempty"`
	SpO2Pct            *string   `json:"spo2_pct,omitempty" yaml:"spo2_pct,omitempty"`
	SkinTempCelsius    *float64  `json:"skin_temp_celsius,omitempty" yaml:"skin_temp_celsius,omitempty"`
	SkinTempFahrenheit *float64  `json:"skin_temp_fahrenheit,omitempty" yaml:"skin_temp_fahrenheit,omitempty"`
}

type sleepSummary struct {
	ID                         string    `json:"id" yaml:"id"`
	CycleID                    int64     `json:"cycle_id" yaml:"cycle_id"`
	Start                      time.Time `json:"start" yaml:"start"`
	StartLocalDate             string    `json:"start_local_date" yaml:"start_local_date"`
	StartLocalTime             string    `json:"start_local_time" yaml:"start_local_time"`
	End                        time.Time `json:"end" yaml:"end"`
	EndLocalDate               string    `json:"end_local_date" yaml:"end_local_date"`
	EndLocalTime               string    `json:"end_local_time" yaml:"end_local_time"`
	Duration                   string    `json:"duration" yaml:"duration"`
	TimezoneOffset             string    `json:"timezone_offset" yaml:"timezone_offset"`
	Nap                        bool      `json:"nap" yaml:"nap"`
	ScoreState                 string    `json:"score_state" yaml:"score_state"`
	SleepPerformancePercentage *float64  `json:"sleep_performance_percentage,omitempty" yaml:"sleep_performance_percentage,omitempty"`
	SleepPerformancePct        *string   `json:"sleep_performance_pct,omitempty" yaml:"sleep_performance_pct,omitempty"`
	SleepEfficiencyPercentage  *float64  `json:"sleep_efficiency_percentage,omitempty" yaml:"sleep_efficiency_percentage,omitempty"`
	SleepEfficiencyPct         *string   `json:"sleep_efficiency_pct,omitempty" yaml:"sleep_efficiency_pct,omitempty"`
}

type workoutSummary struct {
	ID                  string              `json:"id" yaml:"id"`
	SportName           string              `json:"sport_name" yaml:"sport_name"`
	Start               time.Time           `json:"start" yaml:"start"`
	StartLocalDate      string              `json:"start_local_date" yaml:"start_local_date"`
	StartLocalTime      string              `json:"start_local_time" yaml:"start_local_time"`
	End                 time.Time           `json:"end" yaml:"end"`
	EndLocalDate        string              `json:"end_local_date" yaml:"end_local_date"`
	EndLocalTime        string              `json:"end_local_time" yaml:"end_local_time"`
	Duration            string              `json:"duration" yaml:"duration"`
	TimezoneOffset      string              `json:"timezone_offset" yaml:"timezone_offset"`
	ScoreState          string              `json:"score_state" yaml:"score_state"`
	Strain              *float64            `json:"strain,omitempty" yaml:"strain,omitempty"`
	Kilojoule           *float64            `json:"kilojoule,omitempty" yaml:"kilojoule,omitempty"`
	Kilocalorie         *float64            `json:"kilocalorie,omitempty" yaml:"kilocalorie,omitempty"`
	AverageHeartRate    *int                `json:"average_heart_rate,omitempty" yaml:"average_heart_rate,omitempty"`
	AverageHRUnit       string              `json:"average_heart_rate_unit,omitempty" yaml:"average_heart_rate_unit,omitempty"`
	MaxHeartRate        *int                `json:"max_heart_rate,omitempty" yaml:"max_heart_rate,omitempty"`
	MaxHRUnit           string              `json:"max_heart_rate_unit,omitempty" yaml:"max_heart_rate_unit,omitempty"`
	DistanceMeter       *float64            `json:"distance_meter,omitempty" yaml:"distance_meter,omitempty"`
	Distance            *string             `json:"distance,omitempty" yaml:"distance,omitempty"`
	DistanceKilometer   *float64            `json:"distance_kilometer,omitempty" yaml:"distance_kilometer,omitempty"`
	DistanceMile        *float64            `json:"distance_mile,omitempty" yaml:"distance_mile,omitempty"`
	AltitudeGainMeter   *float64            `json:"altitude_gain_meter,omitempty" yaml:"altitude_gain_meter,omitempty"`
	AltitudeGain        *string             `json:"altitude_gain,omitempty" yaml:"altitude_gain,omitempty"`
	AltitudeGainFeet    *float64            `json:"altitude_gain_feet,omitempty" yaml:"altitude_gain_feet,omitempty"`
	AltitudeChangeMeter *float64            `json:"altitude_change_meter,omitempty" yaml:"altitude_change_meter,omitempty"`
	AltitudeChange      *string             `json:"altitude_change,omitempty" yaml:"altitude_change,omitempty"`
	AltitudeChangeFeet  *float64            `json:"altitude_change_feet,omitempty" yaml:"altitude_change_feet,omitempty"`
	ZoneDurations       *workoutZonesDetail `json:"zone_durations,omitempty" yaml:"zone_durations,omitempty"`
}

func mapCycles(cycles []whoop.Cycle) []cycleSummary {
	items := make([]cycleSummary, 0, len(cycles))
	for _, cycle := range cycles {
		item := cycleSummary{
			ID:             cycle.ID,
			Start:          cycle.Start,
			StartLocalDate: localDate(cycle.Start),
			StartLocalTime: localTime(cycle.Start),
			End:            cycle.End,
			TimezoneOffset: cycle.TimezoneOffset,
			ScoreState:     string(cycle.ScoreState),
		}
		if cycle.End != nil {
			item.EndLocalDate = new(localDate(*cycle.End))
			item.EndLocalTime = new(localTime(*cycle.End))
			item.Duration = new(formatDuration(cycle.End.Sub(cycle.Start)))
		}
		if cycle.Score != nil {
			item.Strain = &cycle.Score.Strain
			item.Kilojoule = &cycle.Score.Kilojoule
			item.Kilocalorie = new(kilojouleToKilocalorie(cycle.Score.Kilojoule))
			item.AverageHR = &cycle.Score.AverageHeartRate
			item.AverageHRUnit = unitBPM
			item.MaxHR = &cycle.Score.MaxHeartRate
			item.MaxHRUnit = unitBPM
		}
		items = append(items, item)
	}
	return items
}

func mapRecoveries(recoveries []whoop.Recovery) []recoverySummary {
	items := make([]recoverySummary, 0, len(recoveries))
	for _, recovery := range recoveries {
		item := recoverySummary{
			CycleID:    recovery.CycleID,
			SleepID:    recovery.SleepID,
			UpdatedAt:  recovery.UpdatedAt,
			ScoreState: string(recovery.ScoreState),
		}
		if recovery.Score != nil {
			item.RecoveryScore = &recovery.Score.RecoveryScore
			item.RecoveryScorePct = new(formatPercent(recovery.Score.RecoveryScore))
			item.RestingHeartRate = &recovery.Score.RestingHeartRate
			item.RestingHRUnit = unitBPM
			item.HRVRmssdMilli = &recovery.Score.HRVRmssdMilli
			item.HRVRmssd = new(formatMilliseconds(recovery.Score.HRVRmssdMilli))
			item.SpO2Percentage = &recovery.Score.SpO2Percentage
			item.SpO2Pct = new(formatPercent(recovery.Score.SpO2Percentage))
			item.SkinTempCelsius = &recovery.Score.SkinTempCelsius
			item.SkinTempFahrenheit = new(celsiusToFahrenheit(recovery.Score.SkinTempCelsius))
		}
		items = append(items, item)
	}
	return items
}

func mapSleeps(sleeps []whoop.Sleep) []sleepSummary {
	items := make([]sleepSummary, 0, len(sleeps))
	for _, sleep := range sleeps {
		item := sleepSummary{
			ID:             sleep.ID,
			CycleID:        sleep.CycleID,
			Start:          sleep.Start,
			StartLocalDate: localDate(sleep.Start),
			StartLocalTime: localTime(sleep.Start),
			End:            sleep.End,
			EndLocalDate:   localDate(sleep.End),
			EndLocalTime:   localTime(sleep.End),
			Duration:       formatDuration(sleep.End.Sub(sleep.Start)),
			TimezoneOffset: sleep.TimezoneOffset,
			Nap:            sleep.Nap,
			ScoreState:     string(sleep.ScoreState),
		}
		if sleep.Score != nil {
			item.SleepPerformancePercentage = &sleep.Score.SleepPerformancePercentage
			item.SleepPerformancePct = new(formatPercent(sleep.Score.SleepPerformancePercentage))
			item.SleepEfficiencyPercentage = &sleep.Score.SleepEfficiencyPercentage
			item.SleepEfficiencyPct = new(formatPercent(sleep.Score.SleepEfficiencyPercentage))
		}
		items = append(items, item)
	}
	return items
}

func mapWorkouts(workouts []whoop.Workout) []workoutSummary {
	items := make([]workoutSummary, 0, len(workouts))
	for _, workout := range workouts {
		item := workoutSummary{
			ID:             workout.ID,
			SportName:      workout.SportName,
			Start:          workout.Start,
			StartLocalDate: localDate(workout.Start),
			StartLocalTime: localTime(workout.Start),
			End:            workout.End,
			EndLocalDate:   localDate(workout.End),
			EndLocalTime:   localTime(workout.End),
			Duration:       formatDuration(workout.End.Sub(workout.Start)),
			TimezoneOffset: workout.TimezoneOffset,
			ScoreState:     string(workout.ScoreState),
		}
		if workout.Score != nil {
			item.Strain = &workout.Score.Strain
			item.Kilojoule = &workout.Score.Kilojoule
			item.Kilocalorie = new(kilojouleToKilocalorie(workout.Score.Kilojoule))
			item.AverageHeartRate = &workout.Score.AverageHeartRate
			item.AverageHRUnit = unitBPM
			item.MaxHeartRate = &workout.Score.MaxHeartRate
			item.MaxHRUnit = unitBPM
			item.DistanceMeter = workout.Score.DistanceMeter
			if workout.Score.DistanceMeter != nil {
				item.Distance = new(formatMeters(*workout.Score.DistanceMeter))
				item.DistanceKilometer = new(metersToKilometers(*workout.Score.DistanceMeter))
				item.DistanceMile = new(metersToMiles(*workout.Score.DistanceMeter))
			}
			item.AltitudeGainMeter = workout.Score.AltitudeGainMeter
			if workout.Score.AltitudeGainMeter != nil {
				item.AltitudeGain = new(formatMeters(*workout.Score.AltitudeGainMeter))
				item.AltitudeGainFeet = new(metersToFeet(*workout.Score.AltitudeGainMeter))
			}
			item.AltitudeChangeMeter = workout.Score.AltitudeChangeMeter
			if workout.Score.AltitudeChangeMeter != nil {
				item.AltitudeChange = new(formatMeters(*workout.Score.AltitudeChangeMeter))
				item.AltitudeChangeFeet = new(metersToFeet(*workout.Score.AltitudeChangeMeter))
			}
			item.ZoneDurations = mapWorkoutZones(workout.Score.ZoneDurations)
		}
		items = append(items, item)
	}
	return items
}

type sleepDetail struct {
	ID             string            `json:"id" yaml:"id"`
	CycleID        int64             `json:"cycle_id" yaml:"cycle_id"`
	V1ID           *int64            `json:"v1_id,omitempty" yaml:"v1_id,omitempty"`
	UserID         int64             `json:"user_id" yaml:"user_id"`
	CreatedAt      time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" yaml:"updated_at"`
	Start          time.Time         `json:"start" yaml:"start"`
	StartLocalDate string            `json:"start_local_date" yaml:"start_local_date"`
	StartLocalTime string            `json:"start_local_time" yaml:"start_local_time"`
	End            time.Time         `json:"end" yaml:"end"`
	EndLocalDate   string            `json:"end_local_date" yaml:"end_local_date"`
	EndLocalTime   string            `json:"end_local_time" yaml:"end_local_time"`
	Duration       string            `json:"duration" yaml:"duration"`
	TimezoneOffset string            `json:"timezone_offset" yaml:"timezone_offset"`
	Nap            bool              `json:"nap" yaml:"nap"`
	ScoreState     string            `json:"score_state" yaml:"score_state"`
	Score          *sleepScoreDetail `json:"score,omitempty" yaml:"score,omitempty"`
}

type sleepScoreDetail struct {
	StageSummary               sleepStagesDetail `json:"stage_summary" yaml:"stage_summary"`
	SleepNeeded                sleepNeededDetail `json:"sleep_needed" yaml:"sleep_needed"`
	RespiratoryRate            float64           `json:"respiratory_rate" yaml:"respiratory_rate"`
	RespiratoryRateUnit        string            `json:"respiratory_rate_unit" yaml:"respiratory_rate_unit"`
	SleepPerformancePercentage float64           `json:"sleep_performance_percentage" yaml:"sleep_performance_percentage"`
	SleepPerformancePct        string            `json:"sleep_performance_pct" yaml:"sleep_performance_pct"`
	SleepConsistencyPercentage float64           `json:"sleep_consistency_percentage" yaml:"sleep_consistency_percentage"`
	SleepConsistencyPct        string            `json:"sleep_consistency_pct" yaml:"sleep_consistency_pct"`
	SleepEfficiencyPercentage  float64           `json:"sleep_efficiency_percentage" yaml:"sleep_efficiency_percentage"`
	SleepEfficiencyPct         string            `json:"sleep_efficiency_pct" yaml:"sleep_efficiency_pct"`
}

type sleepStagesDetail struct {
	TotalInBedTimeMilli         int    `json:"total_in_bed_time_milli" yaml:"total_in_bed_time_milli"`
	TotalInBedTime              string `json:"total_in_bed_time" yaml:"total_in_bed_time"`
	TotalAwakeTimeMilli         int    `json:"total_awake_time_milli" yaml:"total_awake_time_milli"`
	TotalAwakeTime              string `json:"total_awake_time" yaml:"total_awake_time"`
	TotalNoDataTimeMilli        int    `json:"total_no_data_time_milli" yaml:"total_no_data_time_milli"`
	TotalNoDataTime             string `json:"total_no_data_time" yaml:"total_no_data_time"`
	TotalLightSleepTimeMilli    int    `json:"total_light_sleep_time_milli" yaml:"total_light_sleep_time_milli"`
	TotalLightSleepTime         string `json:"total_light_sleep_time" yaml:"total_light_sleep_time"`
	TotalSlowWaveSleepTimeMilli int    `json:"total_slow_wave_sleep_time_milli" yaml:"total_slow_wave_sleep_time_milli"`
	TotalSlowWaveSleepTime      string `json:"total_slow_wave_sleep_time" yaml:"total_slow_wave_sleep_time"`
	TotalREMSleepTimeMilli      int    `json:"total_rem_sleep_time_milli" yaml:"total_rem_sleep_time_milli"`
	TotalREMSleepTime           string `json:"total_rem_sleep_time" yaml:"total_rem_sleep_time"`
	SleepCycleCount             int    `json:"sleep_cycle_count" yaml:"sleep_cycle_count"`
	DisturbanceCount            int    `json:"disturbance_count" yaml:"disturbance_count"`
}

type sleepNeededDetail struct {
	BaselineMilli             int    `json:"baseline_milli" yaml:"baseline_milli"`
	Baseline                  string `json:"baseline" yaml:"baseline"`
	NeedFromSleepDebtMilli    int    `json:"need_from_sleep_debt_milli" yaml:"need_from_sleep_debt_milli"`
	NeedFromSleepDebt         string `json:"need_from_sleep_debt" yaml:"need_from_sleep_debt"`
	NeedFromRecentStrainMilli int    `json:"need_from_recent_strain_milli" yaml:"need_from_recent_strain_milli"`
	NeedFromRecentStrain      string `json:"need_from_recent_strain" yaml:"need_from_recent_strain"`
	NeedFromRecentNapMilli    int    `json:"need_from_recent_nap_milli" yaml:"need_from_recent_nap_milli"`
	NeedFromRecentNap         string `json:"need_from_recent_nap" yaml:"need_from_recent_nap"`
}

type workoutZonesDetail struct {
	ZoneZeroMilli  int    `json:"zone_zero_milli" yaml:"zone_zero_milli"`
	ZoneZero       string `json:"zone_zero" yaml:"zone_zero"`
	ZoneOneMilli   int    `json:"zone_one_milli" yaml:"zone_one_milli"`
	ZoneOne        string `json:"zone_one" yaml:"zone_one"`
	ZoneTwoMilli   int    `json:"zone_two_milli" yaml:"zone_two_milli"`
	ZoneTwo        string `json:"zone_two" yaml:"zone_two"`
	ZoneThreeMilli int    `json:"zone_three_milli" yaml:"zone_three_milli"`
	ZoneThree      string `json:"zone_three" yaml:"zone_three"`
	ZoneFourMilli  int    `json:"zone_four_milli" yaml:"zone_four_milli"`
	ZoneFour       string `json:"zone_four" yaml:"zone_four"`
	ZoneFiveMilli  int    `json:"zone_five_milli" yaml:"zone_five_milli"`
	ZoneFive       string `json:"zone_five" yaml:"zone_five"`
}

func mapSleepDetail(sleep *whoop.Sleep) any {
	if sleep == nil {
		return nil
	}
	item := sleepDetail{
		ID:             sleep.ID,
		CycleID:        sleep.CycleID,
		V1ID:           sleep.V1ID,
		UserID:         sleep.UserID,
		CreatedAt:      sleep.CreatedAt,
		UpdatedAt:      sleep.UpdatedAt,
		Start:          sleep.Start,
		StartLocalDate: localDate(sleep.Start),
		StartLocalTime: localTime(sleep.Start),
		End:            sleep.End,
		EndLocalDate:   localDate(sleep.End),
		EndLocalTime:   localTime(sleep.End),
		Duration:       formatDuration(sleep.End.Sub(sleep.Start)),
		TimezoneOffset: sleep.TimezoneOffset,
		Nap:            sleep.Nap,
		ScoreState:     string(sleep.ScoreState),
	}
	if sleep.Score != nil {
		stage := sleep.Score.StageSummary
		needed := sleep.Score.SleepNeeded
		item.Score = &sleepScoreDetail{
			StageSummary: sleepStagesDetail{
				TotalInBedTimeMilli:         stage.TotalInBedTimeMilli,
				TotalInBedTime:              formatMilliDuration(stage.TotalInBedTimeMilli),
				TotalAwakeTimeMilli:         stage.TotalAwakeTimeMilli,
				TotalAwakeTime:              formatMilliDuration(stage.TotalAwakeTimeMilli),
				TotalNoDataTimeMilli:        stage.TotalNoDataTimeMilli,
				TotalNoDataTime:             formatMilliDuration(stage.TotalNoDataTimeMilli),
				TotalLightSleepTimeMilli:    stage.TotalLightSleepTimeMilli,
				TotalLightSleepTime:         formatMilliDuration(stage.TotalLightSleepTimeMilli),
				TotalSlowWaveSleepTimeMilli: stage.TotalSlowWaveSleepTimeMilli,
				TotalSlowWaveSleepTime:      formatMilliDuration(stage.TotalSlowWaveSleepTimeMilli),
				TotalREMSleepTimeMilli:      stage.TotalREMSleepTimeMilli,
				TotalREMSleepTime:           formatMilliDuration(stage.TotalREMSleepTimeMilli),
				SleepCycleCount:             stage.SleepCycleCount,
				DisturbanceCount:            stage.DisturbanceCount,
			},
			SleepNeeded: sleepNeededDetail{
				BaselineMilli:             needed.BaselineMilli,
				Baseline:                  formatMilliDuration(needed.BaselineMilli),
				NeedFromSleepDebtMilli:    needed.NeedFromSleepDebtMilli,
				NeedFromSleepDebt:         formatMilliDuration(needed.NeedFromSleepDebtMilli),
				NeedFromRecentStrainMilli: needed.NeedFromRecentStrainMilli,
				NeedFromRecentStrain:      formatMilliDuration(needed.NeedFromRecentStrainMilli),
				NeedFromRecentNapMilli:    needed.NeedFromRecentNapMilli,
				NeedFromRecentNap:         formatMilliDuration(needed.NeedFromRecentNapMilli),
			},
			RespiratoryRate:            sleep.Score.RespiratoryRate,
			RespiratoryRateUnit:        "breaths_per_min",
			SleepPerformancePercentage: sleep.Score.SleepPerformancePercentage,
			SleepPerformancePct:        formatPercent(sleep.Score.SleepPerformancePercentage),
			SleepConsistencyPercentage: sleep.Score.SleepConsistencyPercentage,
			SleepConsistencyPct:        formatPercent(sleep.Score.SleepConsistencyPercentage),
			SleepEfficiencyPercentage:  sleep.Score.SleepEfficiencyPercentage,
			SleepEfficiencyPct:         formatPercent(sleep.Score.SleepEfficiencyPercentage),
		}
	}
	return item
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%g%%", value)
}

func formatMilliseconds(value float64) string {
	return fmt.Sprintf("%g ms", value)
}

func localDate(value time.Time) string {
	return value.Format("2006-01-02")
}

func localTime(value time.Time) string {
	return value.Format("15:04")
}

func kilojouleToKilocalorie(value float64) float64 {
	return round2(value / 4.184)
}

func celsiusToFahrenheit(value float64) float64 {
	return round2(value*9/5 + 32)
}

func metersToKilometers(value float64) float64 {
	return round2(value / 1000)
}

func metersToMiles(value float64) float64 {
	return round2(value / 1609.344)
}

func metersToFeet(value float64) float64 {
	return round2(value * 3.280839895)
}

func formatMeters(value float64) string {
	return fmt.Sprintf("%g m", round2(value))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func mapWorkoutZones(zones whoop.WorkoutZones) *workoutZonesDetail {
	return &workoutZonesDetail{
		ZoneZeroMilli:  zones.ZoneZeroMilli,
		ZoneZero:       formatMilliDuration(zones.ZoneZeroMilli),
		ZoneOneMilli:   zones.ZoneOneMilli,
		ZoneOne:        formatMilliDuration(zones.ZoneOneMilli),
		ZoneTwoMilli:   zones.ZoneTwoMilli,
		ZoneTwo:        formatMilliDuration(zones.ZoneTwoMilli),
		ZoneThreeMilli: zones.ZoneThreeMilli,
		ZoneThree:      formatMilliDuration(zones.ZoneThreeMilli),
		ZoneFourMilli:  zones.ZoneFourMilli,
		ZoneFour:       formatMilliDuration(zones.ZoneFourMilli),
		ZoneFiveMilli:  zones.ZoneFiveMilli,
		ZoneFive:       formatMilliDuration(zones.ZoneFiveMilli),
	}
}

func formatMilliDuration(milliseconds int) string {
	return formatDuration(time.Duration(milliseconds) * time.Millisecond)
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		duration = -duration
	}
	duration = duration.Round(time.Minute)
	if duration == 0 {
		return "0m"
	}

	hours := int(duration / time.Hour)
	duration -= time.Duration(hours) * time.Hour
	minutes := int(duration / time.Minute)

	switch {
	case hours > 0 && minutes > 0:
		return fmt.Sprintf("%dh%dm", hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}
