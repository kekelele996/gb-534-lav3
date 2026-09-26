package model

import "time"

// PhaseDoubtNote 保存某个分析阶段当前最新的疑点说明，随分析详情返回。
// 只记录复核/调查结论，不改动原始评分、输入哈希与冻结快照。
type PhaseDoubtNote struct {
	ID                  uint                `gorm:"primaryKey" json:"id"`
	DeviationAnalysisID uint                `gorm:"not null;uniqueIndex:idx_phase_doubt_analysis_phase" json:"deviation_analysis_id"`
	Phase               string              `gorm:"size:24;not null;uniqueIndex:idx_phase_doubt_analysis_phase" json:"phase"`
	Content             string              `gorm:"type:text;not null" json:"content"`
	Status              string              `gorm:"size:24;not null;index" json:"status"`
	FirstNotedAt        time.Time           `gorm:"not null" json:"first_noted_at"`
	FirstNotedBy        uint                `gorm:"not null" json:"first_noted_by"`
	FirstNotedByName    string              `gorm:"size:80;not null" json:"first_noted_by_name"`
	LatestUpdateAt      time.Time           `gorm:"not null" json:"latest_update_at"`
	UpdatedBy           uint                `gorm:"not null" json:"updated_by"`
	UpdatedByName       string              `gorm:"size:80;not null" json:"updated_by_name"`
	Revision            int                 `gorm:"not null;default:1" json:"revision"`
	CreatedAt           time.Time           `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time           `gorm:"not null" json:"updated_at"`
	History             []PhaseDoubtNoteHistory `gorm:"foreignKey:PhaseDoubtNoteID;references:ID" json:"history,omitempty"`
}

func (PhaseDoubtNote) TableName() string { return "phase_doubt_notes" }

// PhaseDoubtNoteHistory 追加保存每次更新的操作者与时间，只增不改，作废分析后不再写入。
type PhaseDoubtNoteHistory struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	PhaseDoubtNoteID    uint      `gorm:"not null;index" json:"phase_doubt_note_id"`
	DeviationAnalysisID uint      `gorm:"not null;index" json:"deviation_analysis_id"`
	Phase               string    `gorm:"size:24;not null;index" json:"phase"`
	Content             string    `gorm:"type:text;not null" json:"content"`
	Status              string    `gorm:"size:24;not null" json:"status"`
	Revision            int       `gorm:"not null" json:"revision"`
	RecordedAt          time.Time `gorm:"not null" json:"recorded_at"`
	RecordedBy          uint      `gorm:"not null" json:"recorded_by"`
	RecordedByName      string    `gorm:"size:80;not null" json:"recorded_by_name"`
	CreatedAt           time.Time `gorm:"not null" json:"created_at"`
}

func (PhaseDoubtNoteHistory) TableName() string { return "phase_doubt_note_histories" }
