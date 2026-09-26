<script setup lang="ts">
import { computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { AlertTriangle, CheckCircle2, History, PencilLine, Save } from 'lucide-vue-next'
import { useAuth } from '../../hooks/useAuth'
import { useAnalysisStore } from '../../stores/deviation-analysis'
import type { DeviationAnalysis, PhaseDoubtNote } from '../../types/deviation-analysis'
import PhaseBadge from './PhaseBadge.vue'

const props = defineProps<{ analysis: DeviationAnalysis | null }>()
const analyses = useAnalysisStore()
const { canReview } = useAuth()

interface PhaseDraft {
  editing: boolean
  status: 'open' | 'resolved'
  note: string
  showHistory: boolean
  saving: boolean
}

const drafts = reactive<Record<string, PhaseDraft>>({})
const notesByPhase = computed(() => {
  const map = new Map<string, PhaseDoubtNote>()
  for (const note of props.analysis?.phase_doubt_notes ?? []) map.set(note.phase, note)
  return map
})
const phases = computed(() => (props.analysis?.phase_scores_json ?? []).map((score) => score.phase))
const voided = computed(() => props.analysis?.analysis_state === 'voided')
const canEdit = computed(() => Boolean(canReview.value && props.analysis && !voided.value))

function draft(phase: string): PhaseDraft {
  if (!drafts[phase]) drafts[phase] = { editing: false, status: 'open', note: '', showHistory: false, saving: false }
  return drafts[phase]
}
function startEdit(phase: string) {
  const existing = notesByPhase.value.get(phase)
  const state = draft(phase)
  state.status = existing?.status === 'resolved' ? 'resolved' : 'open'
  state.note = existing?.note ?? ''
  state.editing = true
}
async function submit(phase: string) {
  const state = draft(phase)
  const note = state.note.trim()
  if (!note) { ElMessage.warning('请填写疑点说明内容'); return }
  state.saving = true
  try {
    await analyses.saveDoubtNote(phase, state.status, note)
    state.editing = false
    ElMessage.success('阶段疑点说明已保存')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '疑点说明保存失败')
  } finally {
    state.saving = false
  }
}
function formatTime(value: string) { return new Date(value).toLocaleString() }
</script>

<template>
  <section v-if="analysis" class="phase-doubt-list">
    <div class="doubt-heading">
      <h3>阶段疑点说明</h3>
      <p>按阶段记录复盘疑点；同一阶段再次提交保留最新内容与最早记录时间，并留下每次更新的操作者。</p>
    </div>
    <el-alert v-if="voided" type="warning" :closable="false" show-icon
      title="该分析结果已作废，疑点记录与评分、输入指纹一并冻结，不再接受修改。" />
    <article v-for="phase in phases" :key="phase" class="phase-doubt-card">
      <header>
        <PhaseBadge :phase="phase" />
        <span v-if="notesByPhase.get(phase)?.status === 'resolved'" class="doubt-status resolved"><CheckCircle2 :size="13" />已澄清</span>
        <span v-else-if="notesByPhase.has(phase)" class="doubt-status open"><AlertTriangle :size="13" />待跟进</span>
        <span v-else class="doubt-status empty">未记录疑点</span>
      </header>
      <template v-if="notesByPhase.has(phase)">
        <p class="doubt-text">{{ notesByPhase.get(phase)?.note }}</p>
        <dl class="doubt-meta">
          <div><dt>最早记录</dt><dd>{{ formatTime(notesByPhase.get(phase)!.first_recorded_at) }} · {{ notesByPhase.get(phase)?.created_by_name }}</dd></div>
          <div><dt>最近更新</dt><dd>{{ formatTime(notesByPhase.get(phase)!.updated_at) }} · {{ notesByPhase.get(phase)?.updated_by_name }}</dd></div>
        </dl>
        <el-tooltip :content="draft(phase).showHistory ? '收起更新记录' : '查看每次更新的操作者与时间'">
          <button type="button" class="doubt-history-toggle" @click="draft(phase).showHistory = !draft(phase).showHistory">
            <History :size="13" />更新记录（{{ notesByPhase.get(phase)?.revisions.length ?? 0 }}）
          </button>
        </el-tooltip>
        <ol v-if="draft(phase).showHistory" class="doubt-history">
          <li v-for="revision in notesByPhase.get(phase)?.revisions ?? []" :key="revision.id">
            <span :class="['doubt-status', revision.status === 'resolved' ? 'resolved' : 'open']">
              <component :is="revision.status === 'resolved' ? CheckCircle2 : AlertTriangle" :size="12" />
              {{ revision.status === 'resolved' ? '已澄清' : '待跟进' }}
            </span>
            <p>{{ revision.note }}</p>
            <small>{{ formatTime(revision.recorded_at) }} · {{ revision.actor_name }}</small>
          </li>
        </ol>
      </template>
      <p v-else class="doubt-empty">该阶段尚无复盘疑点说明。</p>
      <template v-if="canEdit">
        <el-button v-if="!draft(phase).editing" size="small" text type="primary" @click="startEdit(phase)">
          <PencilLine :size="14" />{{ notesByPhase.has(phase) ? '更新疑点说明' : '记录疑点' }}
        </el-button>
        <div v-else class="doubt-form">
          <el-input v-model="draft(phase).note" type="textarea" :rows="3" maxlength="1000" show-word-limit
            :placeholder="`填写 ${phase} 阶段的复盘疑点说明`" />
          <div class="doubt-form-actions">
            <el-select v-model="draft(phase).status" style="width: 128px">
              <el-option label="待跟进" value="open" />
              <el-option label="已澄清" value="resolved" />
            </el-select>
            <el-button size="small" @click="draft(phase).editing = false">取消</el-button>
            <el-button size="small" type="primary" :loading="draft(phase).saving" @click="submit(phase)"><Save :size="14" />保存</el-button>
          </div>
        </div>
      </template>
    </article>
  </section>
</template>
