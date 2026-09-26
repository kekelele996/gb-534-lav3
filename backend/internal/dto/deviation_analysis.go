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
type PhaseDoubtNoteRequest struct {
	Phase   string `json:"phase" binding:"required,oneof=lag growth production harvest"`
	Status  string `json:"status" binding:"required,oneof=pending_followup clarified"`
	Content string `json:"content" binding:"required,max=1000"`
}
type PhaseDoubtNoteHistoryResponse struct {
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	Revision       int       `json:"revision"`
	RecordedAt     time.Time `json:"recorded_at"`
	RecordedBy     uint      `json:"recorded_by"`
	RecordedByName string    `json:"recorded_by_name"`
}
type PhaseDoubtNoteResponse struct {
	Phase            string                         `json:"phase"`
	Content          string                         `json:"content"`
	Status           string                         `json:"status"`
	FirstNotedAt     time.Time                      `json:"first_noted_at"`
	FirstNotedBy     uint                           `json:"first_noted_by"`
	FirstNotedByName string                         `json:"first_noted_by_name"`
	LatestUpdateAt   time.Time                      `json:"latest_update_at"`
	UpdatedBy        uint                           `json:"updated_by"`
	UpdatedByName    string                         `json:"updated_by_name"`
	Revision         int                            `json:"revision"`
	History          []PhaseDoubtNoteHistoryResponse `json:"history"`
}
func NewPhaseDoubtNoteResponse(note model.PhaseDoubtNote) PhaseDoubtNoteResponse {
	response := PhaseDoubtNoteResponse{
		Phase: note.Phase, Content: note.Content, Status: note.Status,
		FirstNotedAt: note.FirstNotedAt, FirstNotedBy: note.FirstNotedBy, FirstNotedByName: note.FirstNotedByName,
		LatestUpdateAt: note.LatestUpdateAt, UpdatedBy: note.UpdatedBy, UpdatedByName: note.UpdatedByName,
		Revision: note.Revision, History: make([]PhaseDoubtNoteHistoryResponse, 0, len(note.History)),
	}
	for _, entry := range note.History {
		response.History = append(response.History, PhaseDoubtNoteHistoryResponse{
			Content: entry.Content, Status: entry.Status, Revision: entry.Revision,
			RecordedAt: entry.RecordedAt, RecordedBy: entry.RecordedBy, RecordedByName: entry.RecordedByName,
		})
	}
	return response
}
type DeviationAnalysisQuery struct {
	SensorSeriesID, RecipeID uint
	State, Level, Initiator  string
	Page, PageSize           int
}
type DeviationAnalysisResponse struct {
	ID                   uint                  `json:"id"`
	SensorSeriesID       uint                  `json:"sensor_series_id"`
	RecipeID             uint                  `json:"recipe_id"`
	RecipeVersion        int                   `json:"recipe_version"`
	AlgorithmVersion     string                `json:"algorithm_version"`
	InputHash            string                `json:"input_hash"`
	PhaseScoresJSON      json.RawMessage       `json:"phase_scores_json"`
	DeviationLevel       string                `json:"deviation_level"`
	AlignedCurveJSON     json.RawMessage       `json:"aligned_curve_json"`
	SuspectedCausesJSON  json.RawMessage       `json:"suspected_causes_json"`
	AnalysisState        string                `json:"analysis_state"`
	Explanation          string                `json:"explanation"`
	AnalyzedAt           time.Time             `json:"analyzed_at"`
	InitiatedBy          uint                  `json:"initiated_by"`
	InitiatedByName      string                `json:"initiated_by_name"`
	ReviewedBy           *uint                 `json:"reviewed_by,omitempty"`
	ReviewedByName       string                `json:"reviewed_by_name,omitempty"`
	DurationMilliseconds int64                 `json:"duration_milliseconds"`
	FailureReason        string                `json:"failure_reason,omitempty"`
	ReviewComment        string                `json:"review_comment,omitempty"`
	ReplayVerified       *bool                 `json:"replay_verified,omitempty"`
	SensorSeries         *SensorSeriesResponse `json:"sensor_series,omitempty"`
	PhaseDoubtNotes      []PhaseDoubtNoteResponse `json:"phase_doubt_notes"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
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
		response.PhaseDoubtNotes = append(response.PhaseDoubtNotes, NewPhaseDoubtNoteResponse(note))
	}
	return response
}
