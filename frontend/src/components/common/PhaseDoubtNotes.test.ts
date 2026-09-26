import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '../../stores/auth'
import type { DeviationAnalysis } from '../../types/deviation-analysis'
import PhaseDoubtNotes from './PhaseDoubtNotes.vue'

function analysisFixture(overrides: Partial<DeviationAnalysis> = {}): DeviationAnalysis {
  return {
    id: 7, sensor_series_id: 3, recipe_id: 2, recipe_version: 1, algorithm_version: 'phase-dtw-v1.0.0',
    input_hash: 'abc123',
    phase_scores_json: [
      { phase: 'lag', duration_deviation: 0, slope_deviation: 0, peak_time_deviation: 0, curve_distance: 0, weighted_deviation: 0.1, channel_scores: {}, observed_points: 3 },
      { phase: 'growth', duration_deviation: 0, slope_deviation: 0, peak_time_deviation: 0, curve_distance: 0, weighted_deviation: 0.5, channel_scores: {}, observed_points: 3 },
    ],
    deviation_level: 'major', aligned_curve_json: [], suspected_causes_json: [],
    analysis_state: 'reviewed', explanation: 'evidence', analyzed_at: '2026-09-26T08:00:00Z',
    initiated_by: 9, initiated_by_name: 'analyst', duration_milliseconds: 4,
    phase_doubt_notes: [
      {
        id: 1, analysis_id: 7, phase: 'growth', status: 'open', note: '生长期斜率异常，待跟进加料记录',
        created_by: 10, created_by_name: 'reviewer', first_recorded_at: '2026-09-26T09:00:00Z',
        updated_by: 10, updated_by_name: 'reviewer', updated_at: '2026-09-26T09:00:00Z',
        revisions: [
          { id: 1, note_id: 1, analysis_id: 7, phase: 'growth', status: 'open', note: '生长期斜率异常，待跟进加料记录', actor_id: 10, actor_name: 'reviewer', recorded_at: '2026-09-26T09:00:00Z' },
        ],
      },
    ],
    created_at: '2026-09-26T08:00:00Z', updated_at: '2026-09-26T09:00:00Z',
    ...overrides,
  }
}

const stubs = {
  'el-alert': { props: ['title'], template: '<div class="alert">{{ title }}</div>' },
  'el-button': { template: '<button :disabled="$attrs.disabled"><slot /></button>' },
  'el-tooltip': { template: '<span><slot /></span>' },
  'el-input': { template: '<textarea />' },
  'el-select': { template: '<select><slot /></select>' },
  'el-option': { template: '<option />' },
}

describe('PhaseDoubtNotes', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders phase doubt notes with status, latest content, earliest record and revisions', () => {
    const auth = useAuthStore()
    auth.user = { id: 10, username: 'reviewer', display_name: 'Reviewer', role: 'reviewer' }
    const wrapper = mount(PhaseDoubtNotes, {
      props: { modelValue: true, analysis: analysisFixture() },
      global: { stubs },
    })
    const text = wrapper.text()
    expect(text).toContain('生长期斜率异常，待跟进加料记录')
    expect(text).toContain('待跟进')
    expect(text).toContain('更新记录（1）')
    expect(text).toContain('该阶段尚无复盘疑点说明')
    expect(text).toContain('记录疑点')
    expect(text).toContain('reviewer')
  })

  it('hides editing controls for roles without review permission', () => {
    const auth = useAuthStore()
    auth.user = { id: 5, username: 'auditor', display_name: 'Auditor', role: 'auditor' }
    const wrapper = mount(PhaseDoubtNotes, {
      props: { modelValue: true, analysis: analysisFixture() },
      global: { stubs },
    })
    const buttons = wrapper.findAll('button').map((button) => button.text())
    expect(buttons.some((label) => label.includes('记录疑点'))).toBe(false)
    expect(buttons.some((label) => label.includes('更新疑点说明'))).toBe(false)
  })

  it('freezes notes and hides editing controls when the analysis is voided', () => {
    const auth = useAuthStore()
    auth.user = { id: 10, username: 'reviewer', display_name: 'Reviewer', role: 'reviewer' }
    const wrapper = mount(PhaseDoubtNotes, {
      props: { modelValue: true, analysis: analysisFixture({ analysis_state: 'voided' }) },
      global: { stubs },
    })
    expect(wrapper.text()).toContain('已作废')
    const buttons = wrapper.findAll('button').map((button) => button.text())
    expect(buttons.some((label) => label.includes('记录疑点'))).toBe(false)
    expect(buttons.some((label) => label.includes('更新疑点说明'))).toBe(false)
  })
})
