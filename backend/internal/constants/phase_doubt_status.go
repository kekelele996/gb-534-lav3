package constants
type PhaseDoubtStatus string
const (
	PhaseDoubtOpen     PhaseDoubtStatus = "open"
	PhaseDoubtResolved PhaseDoubtStatus = "resolved"
)
func (s PhaseDoubtStatus) Valid() bool {
	switch s {
	case PhaseDoubtOpen, PhaseDoubtResolved:
		return true
	default:
		return false
	}
}
func PhaseDoubtStatusValues() []string {
	return []string{string(PhaseDoubtOpen), string(PhaseDoubtResolved)}
}
