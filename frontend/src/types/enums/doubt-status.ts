export type DoubtStatus = 'pending_followup' | 'clarified'

export const doubtStatusLabels: Record<DoubtStatus, string> = {
  pending_followup: '待跟进',
  clarified: '已澄清',
}
