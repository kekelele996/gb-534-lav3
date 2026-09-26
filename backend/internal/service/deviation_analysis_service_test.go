package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/algorithm"
	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/timeseries"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func TestAnalysisIdempotencyReviewerSeparationAndReplay(t *testing.T) {
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	seriesRepo := repository.NewSensorSeriesRepository(db)
	analysisRepo := repository.NewDeviationAnalysisRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-A1", Name: "Analysis vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	boundaries, references, tolerances := testRecipeConfig(t)
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "ANALYSIS-A", Version: 1, Organism: "Test organism",
		TargetDurationH: 8, PhaseBoundariesJSON: string(boundaries), ReferenceCurvesJSON: string(references),
		ToleranceProfileJSON: string(tolerances), RecipeState: "published",
		CreatedBy: 8, CreatedByName: "scientist", CreatedAt: now, UpdatedAt: now,
	}
	if err := recipeRepo.Create(context.Background(), &recipe); err != nil {
		t.Fatal(err)
	}
	points := make([]timeseries.Point, 0, 9)
	for hour := 0; hour <= 8; hour++ {
		value := 7 - float64(hour)*0.05
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: now.Add(time.Duration(hour) * time.Hour), Values: map[string]*float64{"ph": &valueCopy},
		})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	series := model.SensorSeries{
		VesselID: vessel.ID, RecipeID: recipe.ID, RunCode: "RUN-A1", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, StartedAt: now, EndedAt: now.Add(8 * time.Hour),
		SourceChecksum: util.HashString(pointsJSON), SeriesState: "ready", QualitySummary: `{"valid":true}`,
		NormalizationJSON: `{"method":"median_iqr"}`, ImportedBy: 9, ImportedByName: "analyst",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := seriesRepo.Create(context.Background(), &series); err != nil {
		t.Fatal(err)
	}
	svc := NewDeviationAnalysisService(analysisRepo, recipeRepo, seriesRepo, auditRepo, algorithm.NewEvaluator())
	initiator := util.Actor{UserID: 9, Username: "analyst", Role: "data_analyst", RequestID: "req-run"}
	first, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-a", initiator)
	if err != nil || reused {
		t.Fatalf("first run reused=%v err=%v", reused, err)
	}
	second, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-a", initiator)
	if err != nil || !reused || second.ID != first.ID {
		t.Fatalf("same-key run id=%d reused=%v err=%v", second.ID, reused, err)
	}
	third, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-b", initiator)
	if err != nil || !reused || third.ID != first.ID {
		t.Fatalf("same-input run id=%d reused=%v err=%v", third.ID, reused, err)
	}
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-review"}
	if _, err := svc.Transition(context.Background(), first.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "reviewed", Comment: "Evidence reviewed.",
	}, reviewer); err != nil {
		t.Fatalf("review transition: %v", err)
	}
	_, err = svc.Transition(context.Background(), first.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "confirmed", Comment: "Self confirmation must fail.",
	}, initiator)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != util.CodeReviewerConflict {
		t.Fatalf("self-confirm error=%v, want reviewer conflict", err)
	}
	if _, err := svc.Transition(context.Background(), first.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "confirmed", Comment: "Independent confirmation.",
	}, reviewer); err != nil {
		t.Fatalf("independent confirm: %v", err)
	}
	replayed, err := svc.Replay(context.Background(), first.ID, reviewer)
	if err != nil || replayed.ReplayVerified == nil || !*replayed.ReplayVerified {
		t.Fatalf("replay verified=%v err=%v", replayed.ReplayVerified, err)
	}
}

func TestRunRequiresReadySeriesAndIdempotencyKey(t *testing.T) {
	db := newTestDB(t)
	svc := NewDeviationAnalysisService(
		repository.NewDeviationAnalysisRepository(db), repository.NewCultureRecipeRepository(db),
		repository.NewSensorSeriesRepository(db), repository.NewAuditRepository(db), algorithm.NewEvaluator(),
	)
	_, _, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: 99}, "", util.Actor{})
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != util.CodeIdempotency {
		t.Fatalf("missing key error=%v", err)
	}
}

func TestAnalysisRoleContract(t *testing.T) {
	if constants.HasPermission(constants.RoleDataAnalyst, constants.PermissionAnalysisConfirm) {
		t.Fatal("data analyst should not receive confirm permission")
	}
}

func TestPhaseDoubtNotesKeepLatestContentEarliestTimeAndHistory(t *testing.T) {
	svc, firstID := prepareCompletedAnalysis(t)
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-note-1"}
	created, err := svc.SavePhaseDoubtNote(context.Background(), firstID, dto.PhaseDoubtNoteRequest{
		Phase: "growth", Status: "pending_followup", Content: "Growth phase DO drop needs follow-up.",
	}, reviewer)
	if err != nil {
		t.Fatalf("add doubt note: %v", err)
	}
	firstNote := findDoubtNote(t, created.PhaseDoubtNotes, "growth")
	if firstNote.Revision != 1 || firstNote.Status != "pending_followup" {
		t.Fatalf("initial note revision=%d status=%s", firstNote.Revision, firstNote.Status)
	}
	if firstNote.FirstNotedByName != "reviewer" || firstNote.UpdatedByName != "reviewer" {
		t.Fatalf("note actors: first=%s latest=%s", firstNote.FirstNotedByName, firstNote.UpdatedByName)
	}
	if len(firstNote.History) != 1 || firstNote.History[0].RecordedByName != "reviewer" {
		t.Fatalf("initial history entries=%d", len(firstNote.History))
	}
	frozenScores := created.PhaseScoresJSON
	frozenHash := created.InputHash
	scientist := util.Actor{UserID: 11, Username: "scientist", Role: "process_scientist", RequestID: "req-note-2"}
	updated, err := svc.SavePhaseDoubtNote(context.Background(), firstID, dto.PhaseDoubtNoteRequest{
		Phase: "growth", Status: "clarified", Content: "Confirmed as probe calibration drift.",
	}, scientist)
	if err != nil {
		t.Fatalf("update doubt note: %v", err)
	}
	latest := findDoubtNote(t, updated.PhaseDoubtNotes, "growth")
	if latest.Content != "Confirmed as probe calibration drift." || latest.Status != "clarified" {
		t.Fatalf("latest content=%q status=%q", latest.Content, latest.Status)
	}
	if !latest.FirstNotedAt.Equal(firstNote.FirstNotedAt) || latest.FirstNotedBy != firstNote.FirstNotedBy {
		t.Fatalf("first noted time/actor changed: %v %d vs %v %d",
			latest.FirstNotedAt, latest.FirstNotedBy, firstNote.FirstNotedAt, firstNote.FirstNotedBy)
	}
	if latest.Revision != 2 || len(latest.History) != 2 {
		t.Fatalf("revision=%d history=%d, want 2/2", latest.Revision, len(latest.History))
	}
	if latest.History[0].Content != "Growth phase DO drop needs follow-up." ||
		latest.History[1].Content != "Confirmed as probe calibration drift." {
		t.Fatalf("history is not append-only in order: %+v", latest.History)
	}
	if latest.UpdatedByName != "scientist" || !latest.LatestUpdateAt.After(latest.FirstNotedAt) && !latest.LatestUpdateAt.Equal(latest.FirstNotedAt) {
		t.Fatalf("latest update actor=%s time=%v", latest.UpdatedByName, latest.LatestUpdateAt)
	}
	if string(updated.PhaseScoresJSON) != string(frozenScores) || updated.InputHash != frozenHash {
		t.Fatal("doubt note changed frozen phase scores or input hash")
	}
	if _, err := svc.Transition(context.Background(), firstID, dto.DeviationAnalysisTransitionRequest{
		ToState: "reviewed", Comment: "Reviewed before void.",
	}, reviewer); err != nil {
		t.Fatalf("review transition: %v", err)
	}
	if _, err := svc.Transition(context.Background(), firstID, dto.DeviationAnalysisTransitionRequest{
		ToState: "voided", Comment: "Void for test.",
	}, reviewer); err != nil {
		t.Fatalf("void transition: %v", err)
	}
	_, err = svc.SavePhaseDoubtNote(context.Background(), firstID, dto.PhaseDoubtNoteRequest{
		Phase: "growth", Status: "pending_followup", Content: "Must be rejected after void.",
	}, scientist)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != util.CodeStateTransition {
		t.Fatalf("post-void update error=%v, want state conflict", err)
	}
	voided, err := svc.Get(context.Background(), firstID)
	if err != nil {
		t.Fatalf("load voided analysis: %v", err)
	}
	afterVoid := findDoubtNote(t, voided.PhaseDoubtNotes, "growth")
	if afterVoid.Content != "Confirmed as probe calibration drift." || afterVoid.Revision != 2 {
		t.Fatalf("voided analysis note changed: content=%q revision=%d", afterVoid.Content, afterVoid.Revision)
	}
}

func TestPhaseDoubtNoteRejectsBlankAndUnknownFields(t *testing.T) {
	svc, firstID := prepareCompletedAnalysis(t)
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-note-invalid"}
	_, err := svc.SavePhaseDoubtNote(context.Background(), firstID, dto.PhaseDoubtNoteRequest{
		Phase: "growth", Status: "pending_followup", Content: "   ",
	}, reviewer)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != util.CodeValidation {
		t.Fatalf("blank content error=%v, want validation", err)
	}
	_, err = svc.SavePhaseDoubtNote(context.Background(), firstID, dto.PhaseDoubtNoteRequest{
		Phase: "unknown", Status: "pending_followup", Content: "Invalid phase.",
	}, reviewer)
	if !errors.As(err, &appErr) || appErr.Code != util.CodeValidation {
		t.Fatalf("unknown phase error=%v, want validation", err)
	}
}

func prepareCompletedAnalysis(t *testing.T) (*DeviationAnalysisService, uint) {
	t.Helper()
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	seriesRepo := repository.NewSensorSeriesRepository(db)
	analysisRepo := repository.NewDeviationAnalysisRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-N1", Name: "Note vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	boundaries, references, tolerances := testRecipeConfig(t)
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "ANALYSIS-N", Version: 1, Organism: "Test organism",
		TargetDurationH: 8, PhaseBoundariesJSON: string(boundaries), ReferenceCurvesJSON: string(references),
		ToleranceProfileJSON: string(tolerances), RecipeState: "published",
		CreatedBy: 8, CreatedByName: "scientist", CreatedAt: now, UpdatedAt: now,
	}
	if err := recipeRepo.Create(context.Background(), &recipe); err != nil {
		t.Fatal(err)
	}
	points := make([]timeseries.Point, 0, 9)
	for hour := 0; hour <= 8; hour++ {
		value := 7 - float64(hour)*0.05
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: now.Add(time.Duration(hour) * time.Hour), Values: map[string]*float64{"ph": &valueCopy},
		})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	series := model.SensorSeries{
		VesselID: vessel.ID, RecipeID: recipe.ID, RunCode: "RUN-N1", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, StartedAt: now, EndedAt: now.Add(8 * time.Hour),
		SourceChecksum: util.HashString(pointsJSON), SeriesState: "ready", QualitySummary: `{"valid":true}`,
		NormalizationJSON: `{"method":"median_iqr"}`, ImportedBy: 9, ImportedByName: "analyst",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := seriesRepo.Create(context.Background(), &series); err != nil {
		t.Fatal(err)
	}
	svc := NewDeviationAnalysisService(analysisRepo, recipeRepo, seriesRepo, auditRepo, algorithm.NewEvaluator())
	initiator := util.Actor{UserID: 9, Username: "analyst", Role: "data_analyst", RequestID: "req-run-note"}
	analysis, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-note", initiator)
	if err != nil || reused {
		t.Fatalf("run analysis reused=%v err=%v", reused, err)
	}
	return svc, analysis.ID
}

func findDoubtNote(t *testing.T, notes []dto.PhaseDoubtNoteResponse, phase string) dto.PhaseDoubtNoteResponse {
	t.Helper()
	for _, note := range notes {
		if note.Phase == phase {
			return note
		}
	}
	t.Fatalf("doubt note for phase %s not found", phase)
	return dto.PhaseDoubtNoteResponse{}
}
