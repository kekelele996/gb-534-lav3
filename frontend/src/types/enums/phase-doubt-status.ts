export const phaseDoubtStatuses = ['open', 'resolved'] as const
export type PhaseDoubtStatus = typeof phaseDoubtStatuses[number]
