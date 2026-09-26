package constants

type DoubtStatus string

const (
	DoubtPendingFollowup DoubtStatus = "pending_followup"
	DoubtClarified       DoubtStatus = "clarified"
)

func (s DoubtStatus) Valid() bool {
	switch s {
	case DoubtPendingFollowup, DoubtClarified:
		return true
	default:
		return false
	}
}

func DoubtStatusValues() []string {
	return []string{string(DoubtPendingFollowup), string(DoubtClarified)}
}
