import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../stores/deviation-analysis', () => ({
  useAnalysisStore: () => ({ selected: null, load: vi.fn().mockResolvedValue(undefined) }),
}))
vi.mock('../../api/deviation-analysis', () => ({
  savePhaseDoubtNote: vi.fn().mockResolvedValue({}),
}))
import PhaseDoubtNote from './PhaseDoubtNote.vue'

beforeEach(() => vi.clearAllMocks())

const stubs = {
  'el-button': { template: '<button :disabled="disabled"><slot /></button>', props: ['disabled', 'text', 'type', 'size', 'circle'] },
  'el-tag': { template: '<span class="tag"><slot /></span>', props: ['size', 'type', 'effect', 'disableTransitions'] },
  'el-dialog': { template: '<div v-if="modelValue"><slot /><slot name="footer" /></div>', props: ['modelValue', 'title', 'width', 'appendToBody'] },
  'el-form': { template: '<form><slot /></form>' },
  'el-form-item': { template: '<div><label /><slot /></div>', props: ['label'] },
  'el-radio-group': { template: '<div data-radio-group><slot /></div>' },
  'el-radio': { template: '<label><slot /></label>', props: ['value'] },
  'el-input': { template: '<textarea />', props: ['modelValue', 'type', 'rows', 'maxlength', 'showWordLimit', 'placeholder'] },
}

const note = {
  phase: 'growth',
  content: '生长期溶氧曲线偏移，待核对通气记录',
  status: 'pending_followup' as const,
  first_noted_at: '2026-09-20T02:00:00Z',
  first_noted_by: 4,
  first_noted_by_name: 'reviewer',
  latest_update_at: '2026-09-21T08:30:00Z',
  updated_by: 3,
  updated_by_name: 'scientist',
  revision: 2,
  history: [
    { content: '初版怀疑', status: 'pending_followup' as const, revision: 1, recorded_at: '2026-09-20T02:00:00Z', recorded_by: 4, recorded_by_name: 'reviewer' },
    { content: '生长期溶氧曲线偏移，待核对通气记录', status: 'pending_followup' as const, revision: 2, recorded_at: '2026-09-21T08:30:00Z', recorded_by: 3, recorded_by_name: 'scientist' },
  ],
}

describe('PhaseDoubtNote', () => {
  it('renders the latest content while keeping the first noted time and revision history', () => {
    const wrapper = mount(PhaseDoubtNote, {
      props: { analysisId: 7, phase: 'growth', note, editable: false, voided: false },
      global: { stubs },
    })
    expect(wrapper.text()).toContain('待跟进')
    expect(wrapper.text()).toContain(note.content)
    expect(wrapper.text()).toContain('reviewer')
    expect(wrapper.text()).toContain('scientist')
    expect(wrapper.text()).toContain('第 2 版')
  })

  it('offers an editor only to reviewers and process scientists', () => {
    const locked = mount(PhaseDoubtNote, {
      props: { analysisId: 7, phase: 'growth', note, editable: false, voided: false },
      global: { stubs },
    })
    expect(locked.find('button[aria-label]').exists()).toBe(false)
    const editable = mount(PhaseDoubtNote, {
      props: { analysisId: 7, phase: 'growth', note, editable: true, voided: false },
      global: { stubs },
    })
    expect(editable.find('button[aria-label="编辑growth阶段疑点说明"]').exists()).toBe(true)
  })

  it('explains that voided results no longer accept edits and hides the editor', () => {
    const wrapper = mount(PhaseDoubtNote, {
      props: { analysisId: 7, phase: 'growth', note, editable: false, voided: true },
      global: { stubs },
    })
    expect(wrapper.text()).toContain('已作废 · 只读')
    expect(wrapper.text()).toContain('该结果已作废')
    expect(wrapper.find('button[aria-label]').exists()).toBe(false)
  })
})
