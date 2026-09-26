package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/algorithm"
	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/timeseries"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func preparePhaseDoubtAnalysis(t *testing.T, now time.Time) (*DeviationAnalysisService, model.DeviationAnalysis) {
	t.Helper()
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	seriesRepo := repository.NewSensorSeriesRepository(db)
	analysisRepo := repository.NewDeviationAnalysisRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	vessel := model.FermentationVessel{
		VesselCode: "FV-D1", Name: "Doubt vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	boundaries, references, tolerances := testRecipeConfig(t)
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "DOUBT-A", Version: 1, Organism: "Test organism",
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
		points = append(points, timeseries.Point{
			Timestamp: now.Add(time.Duration(hour) * time.Hour), Values: map[string]*float64{"ph": &value},
		})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	series := model.SensorSeries{
		VesselID: vessel.ID, RecipeID: recipe.ID, RunCode: "RUN-D1", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, StartedAt: now, EndedAt: now.Add(8 * time.Hour),
		SourceChecksum: util.HashString(pointsJSON), SeriesState: "ready", QualitySummary: `{"valid":true}`,
		NormalizationJSON: `{"method":"median_iqr"}`, ImportedBy: 9, ImportedByName: "analyst",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := seriesRepo.Create(context.Background(), &series); err != nil {
		t.Fatal(err)
	}
	svc := NewDeviationAnalysisService(analysisRepo, recipeRepo, seriesRepo, auditRepo, algorithm.NewEvaluator())
	runner := util.Actor{UserID: 9, Username: "analyst", Role: "data_analyst", RequestID: "req-doubt-run"}
	result, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-doubt", runner)
	if err != nil || reused {
		t.Fatalf("run analysis reused=%v err=%v", reused, err)
	}
	analysis, err := analysisRepo.GetByID(context.Background(), result.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	return svc, analysis
}

func TestPhaseDoubtNoteKeepsLatestContentAndEarliestRecord(t *testing.T) {
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	svc, analysis := preparePhaseDoubtAnalysis(t, now)
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-doubt-1"}
	first, err := svc.UpsertPhaseDoubtNote(context.Background(), analysis.ID, dto.UpsertPhaseDoubtNoteRequest{
		Phase: "growth", Status: "open", Note: "生长期斜率异常，待跟进加料记录",
	}, reviewer)
	if err != nil {
		t.Fatalf("create phase doubt note: %v", err)
	}
	if len(first.PhaseDoubtNotes) != 1 {
		t.Fatalf("phase doubt notes = %d, want 1", len(first.PhaseDoubtNotes))
	}
	note := first.PhaseDoubtNotes[0]
	if note.Status != "open" || note.CreatedBy != 10 || note.UpdatedBy != 10 {
		t.Fatalf("unexpected first note: %+v", note)
	}
	if len(note.Revisions) != 1 || note.Revisions[0].ActorID != 10 {
		t.Fatalf("first revision trail = %+v", note.Revisions)
	}
	frozenScores := first.PhaseScoresJSON
	frozenHash := first.InputHash
	if string(frozenScores) == "" || frozenHash == "" {
		t.Fatal("frozen scores and input hash must be present")
	}
	scientist := util.Actor{UserID: 8, Username: "scientist", Role: "process_scientist", RequestID: "req-doubt-2"}
	second, err := svc.UpsertPhaseDoubtNote(context.Background(), analysis.ID, dto.UpsertPhaseDoubtNoteRequest{
		Phase: "growth", Status: "resolved", Note: "  已核对加料泵日志，属短时扰动  ",
	}, scientist)
	if err != nil {
		t.Fatalf("update phase doubt note: %v", err)
	}
	if len(second.PhaseDoubtNotes) != 1 {
		t.Fatalf("same phase must keep a single note, got %d", len(second.PhaseDoubtNotes))
	}
	updated := second.PhaseDoubtNotes[0]
	if updated.Note != "已核对加料泵日志，属短时扰动" || updated.Status != "resolved" {
		t.Fatalf("latest content not stored: %+v", updated)
	}
	if updated.UpdatedBy != 8 || updated.UpdatedByName != "scientist" {
		t.Fatalf("latest operator not stored: %+v", updated)
	}
	if updated.CreatedBy != 10 || !updated.FirstRecordedAt.Equal(note.FirstRecordedAt) {
		t.Fatalf("earliest reporter/time must be preserved: first=%+v updated=%+v", note, updated)
	}
	if len(updated.Revisions) != 2 {
		t.Fatalf("revision trail = %d, want 2", len(updated.Revisions))
	}
	if updated.Revisions[0].Note != "生长期斜率异常，待跟进加料记录" || updated.Revisions[1].Note != "已核对加料泵日志，属短时扰动" {
		t.Fatalf("revision snapshots not preserved in order: %+v", updated.Revisions)
	}
	if string(second.PhaseScoresJSON) != string(frozenScores) || second.InputHash != frozenHash {
		t.Fatal("phase scores and input fingerprint must remain untouched")
	}
}

func TestPhaseDoubtNoteRejectsVoidedAnalysisAndMissingPhaseEvidence(t *testing.T) {
	now := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	svc, analysis := preparePhaseDoubtAnalysis(t, now)
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-doubt-void"}
	_, err := svc.UpsertPhaseDoubtNote(context.Background(), analysis.ID, dto.UpsertPhaseDoubtNoteRequest{
		Phase: "unknown_phase", Status: "open", Note: "证据中不存在的阶段",
	}, reviewer)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("missing phase evidence error=%v, want 409", err)
	}
	_, err = svc.UpsertPhaseDoubtNote(context.Background(), analysis.ID, dto.UpsertPhaseDoubtNoteRequest{
		Phase: "lag", Status: "open", Note: "   ",
	}, reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Fatalf("blank note error=%v, want 400", err)
	}
	if _, err := svc.Transition(context.Background(), analysis.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "reviewed", Comment: "reviewed",
	}, reviewer); err != nil {
		t.Fatalf("review transition: %v", err)
	}
	if _, err := svc.Transition(context.Background(), analysis.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "voided", Comment: "voided",
	}, reviewer); err != nil {
		t.Fatalf("void transition: %v", err)
	}
	_, err = svc.UpsertPhaseDoubtNote(context.Background(), analysis.ID, dto.UpsertPhaseDoubtNoteRequest{
		Phase: "growth", Status: "open", Note: "作废后不应再接受修改",
	}, reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("voided update error=%v, want 409", err)
	}
}
