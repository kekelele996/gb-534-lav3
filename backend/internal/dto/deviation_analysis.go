package dto
import (
	"encoding/json"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"time"
)
type RunDeviationAnalysisRequest struct {
	SensorSeriesID uint `json:"sensor_series_id" binding:"required"`
}
type DeviationAnalysisTransitionRequest struct {
	ToState string `json:"to_state" binding:"required,oneof=reviewed confirmed investigating voided"`
	Comment string `json:"comment" binding:"omitempty,max=1000"`
}
type UpsertPhaseDoubtNoteRequest struct {
	Phase  string `json:"phase" binding:"required,oneof=lag growth production harvest"`
	Status string `json:"status" binding:"required,oneof=open resolved"`
	Note   string `json:"note" binding:"required,min=1,max=1000"`
}
type DeviationAnalysisQuery struct {
	SensorSeriesID, RecipeID uint
	State, Level, Initiator  string
	Page, PageSize           int
}
type PhaseDoubtNoteRevisionResponse struct {
	ID         uint      `json:"id"`
	NoteID     uint      `json:"note_id"`
	AnalysisID uint      `json:"analysis_id"`
	Phase      string    `json:"phase"`
	Status     string    `json:"status"`
	Note       string    `json:"note"`
	ActorID    uint      `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	RecordedAt time.Time `json:"recorded_at"`
}
type PhaseDoubtNoteResponse struct {
	ID              uint                            `json:"id"`
	AnalysisID      uint                            `json:"analysis_id"`
	Phase           string                          `json:"phase"`
	Status          string                          `json:"status"`
	Note            string                          `json:"note"`
	CreatedBy       uint                            `json:"created_by"`
	CreatedByName   string                          `json:"created_by_name"`
	FirstRecordedAt time.Time                       `json:"first_recorded_at"`
	UpdatedBy       uint                            `json:"updated_by"`
	UpdatedByName   string                          `json:"updated_by_name"`
	UpdatedAt       time.Time                       `json:"updated_at"`
	Revisions       []PhaseDoubtNoteRevisionResponse `json:"revisions"`
}
type DeviationAnalysisResponse struct {
	ID                   uint                     `json:"id"`
	SensorSeriesID       uint                     `json:"sensor_series_id"`
	RecipeID             uint                     `json:"recipe_id"`
	RecipeVersion        int                      `json:"recipe_version"`
	AlgorithmVersion     string                   `json:"algorithm_version"`
	InputHash            string                   `json:"input_hash"`
	PhaseScoresJSON      json.RawMessage          `json:"phase_scores_json"`
	DeviationLevel       string                   `json:"deviation_level"`
	AlignedCurveJSON     json.RawMessage          `json:"aligned_curve_json"`
	SuspectedCausesJSON  json.RawMessage          `json:"suspected_causes_json"`
	AnalysisState        string                   `json:"analysis_state"`
	Explanation          string                   `json:"explanation"`
	AnalyzedAt           time.Time                `json:"analyzed_at"`
	InitiatedBy          uint                     `json:"initiated_by"`
	InitiatedByName      string                   `json:"initiated_by_name"`
	ReviewedBy           *uint                    `json:"reviewed_by,omitempty"`
	ReviewedByName       string                   `json:"reviewed_by_name,omitempty"`
	DurationMilliseconds int64                    `json:"duration_milliseconds"`
	FailureReason        string                   `json:"failure_reason,omitempty"`
	ReviewComment        string                   `json:"review_comment,omitempty"`
	ReplayVerified       *bool                    `json:"replay_verified,omitempty"`
	SensorSeries         *SensorSeriesResponse    `json:"sensor_series,omitempty"`
	PhaseDoubtNotes      []PhaseDoubtNoteResponse `json:"phase_doubt_notes"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
}
type DeviationAnalysisListResponse struct {
	Items []DeviationAnalysisResponse `json:"items"`
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"page_size"`
}
func NewDeviationAnalysisResponse(analysis model.DeviationAnalysis) DeviationAnalysisResponse {
	response := DeviationAnalysisResponse{
		ID: analysis.ID, SensorSeriesID: analysis.SensorSeriesID, RecipeID: analysis.RecipeID,
		RecipeVersion: analysis.RecipeVersion, AlgorithmVersion: analysis.AlgorithmVersion,
		InputHash: analysis.InputHash, PhaseScoresJSON: rawJSON(analysis.PhaseScoresJSON),
		DeviationLevel: analysis.DeviationLevel, AlignedCurveJSON: rawJSON(analysis.AlignedCurveJSON),
		SuspectedCausesJSON: rawJSON(analysis.SuspectedCausesJSON), AnalysisState: analysis.AnalysisState,
		Explanation: analysis.Explanation, AnalyzedAt: analysis.AnalyzedAt,
		InitiatedBy: analysis.InitiatedBy, InitiatedByName: analysis.InitiatedByName,
		ReviewedBy: analysis.ReviewedBy, ReviewedByName: analysis.ReviewedByName,
		DurationMilliseconds: analysis.DurationMilliseconds, FailureReason: analysis.FailureReason,
		ReviewComment: analysis.ReviewComment, ReplayVerified: analysis.ReplayVerified,
		CreatedAt: analysis.CreatedAt, UpdatedAt: analysis.UpdatedAt,
	}
	if analysis.SensorSeries.ID != 0 {
		s := NewSensorSeriesResponse(analysis.SensorSeries)
		response.SensorSeries = &s
	}
	response.PhaseDoubtNotes = make([]PhaseDoubtNoteResponse, 0, len(analysis.PhaseDoubtNotes))
	for _, note := range analysis.PhaseDoubtNotes {
		noteResponse := PhaseDoubtNoteResponse{
			ID: note.ID, AnalysisID: note.AnalysisID, Phase: note.Phase, Status: note.Status,
			Note: note.Note, CreatedBy: note.CreatedBy, CreatedByName: note.CreatedByName,
			FirstRecordedAt: note.FirstRecordedAt, UpdatedBy: note.UpdatedBy,
			UpdatedByName: note.UpdatedByName, UpdatedAt: note.UpdatedAt,
			Revisions: make([]PhaseDoubtNoteRevisionResponse, 0, len(note.Revisions)),
		}
		for _, revision := range note.Revisions {
			noteResponse.Revisions = append(noteResponse.Revisions, PhaseDoubtNoteRevisionResponse{
				ID: revision.ID, NoteID: revision.NoteID, AnalysisID: revision.AnalysisID,
				Phase: revision.Phase, Status: revision.Status, Note: revision.Note,
				ActorID: revision.ActorID, ActorName: revision.ActorName, RecordedAt: revision.RecordedAt,
			})
		}
		response.PhaseDoubtNotes = append(response.PhaseDoubtNotes, noteResponse)
	}
	return response
}
