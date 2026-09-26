<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleHelp, History, PencilLine } from 'lucide-vue-next'
import PhaseBadge from './PhaseBadge.vue'
import { savePhaseDoubtNote } from '../../api/deviation-analysis'
import { useAnalysisStore } from '../../stores/deviation-analysis'
import { doubtStatusLabels, type DoubtStatus } from '../../types/enums/doubt-status'
import type { PhaseDoubtNote } from '../../types/deviation-analysis'

const props = defineProps<{ analysisId: number; phase: string; note: PhaseDoubtNote | null; editable: boolean; voided: boolean }>()
const store = useAnalysisStore()
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<{ status: DoubtStatus; content: string }>({ status: 'pending_followup', content: '' })
const isClarified = computed(() => props.note?.status === 'clarified')
const historyOpen = ref(false)

function openEditor() {
  form.status = props.note?.status ?? 'pending_followup'
  form.content = props.note?.content ?? ''
  dialogVisible.value = true
}

async function submit() {
  if (!form.content.trim()) {
    ElMessage.warning('请填写疑点说明内容')
    return
  }
  saving.value = true
  try {
    store.selected = await savePhaseDoubtNote(props.analysisId, props.phase, form.status, form.content.trim())
    await store.load()
    ElMessage.success('阶段疑点说明已保存')
    dialogVisible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function formatTime(value: string) {
  return new Date(value).toLocaleString()
}
</script>

<template>
  <div class="doubt-note">
    <div class="doubt-note-head">
      <span class="doubt-note-title"><CircleHelp :size="14" />疑点说明</span>
      <el-tag v-if="voided" size="small" type="info" effect="plain" disable-transitions>已作废 · 只读</el-tag>
      <el-tag v-else-if="note" size="small" :type="isClarified ? 'success' : 'warning'" effect="plain" disable-transitions>
        {{ doubtStatusLabels[note.status] ?? note.status }}
      </el-tag>
      <el-button
        v-if="editable"
        size="small"
        text
        type="primary"
        :aria-label="`编辑${phase}阶段疑点说明`"
        @click="openEditor"
      >
        <PencilLine :size="13" />{{ note ? '更新' : '填写' }}
      </el-button>
    </div>
    <p v-if="note" class="doubt-note-content">{{ note.content }}</p>
    <p v-if="voided" class="doubt-note-empty">该结果已作废，不接受疑点说明修改，记录仅供追踪。</p>
    <p v-else-if="!note && editable" class="doubt-note-empty">尚未记录疑点，可在复核时填写并标注待跟进或已澄清。</p>
    <p v-else-if="!note" class="doubt-note-empty">暂无阶段疑点说明。</p>
    <template v-if="note">
      <div class="doubt-note-meta">
        <span>最早记录：{{ formatTime(note.first_noted_at) }} · {{ note.first_noted_by_name }}</span>
        <span>最近更新：{{ formatTime(note.latest_update_at) }} · {{ note.updated_by_name }} · 第 {{ note.revision }} 版</span>
        <el-button v-if="note.history.length > 1" size="small" text @click="historyOpen = !historyOpen">
          <History :size="12" />{{ historyOpen ? '收起更新记录' : `更新记录（${note.history.length}）` }}
        </el-button>
      </div>
      <ul v-if="historyOpen" class="doubt-history">
        <li v-for="entry in [...note.history].reverse()" :key="entry.revision">
          <el-tag size="small" :type="entry.status === 'clarified' ? 'success' : 'warning'" effect="plain" disable-transitions>
            {{ doubtStatusLabels[entry.status as DoubtStatus] ?? entry.status }}
          </el-tag>
          <span class="doubt-history-text">{{ entry.content }}</span>
          <small>v{{ entry.revision }} · {{ formatTime(entry.recorded_at) }} · {{ entry.recorded_by_name }}</small>
        </li>
      </ul>
    </template>
    <el-dialog
      v-model="dialogVisible"
      :title="`阶段疑点说明`"
      width="min(520px, 92vw)"
      append-to-body
    >
      <div class="doubt-dialog-phase"><PhaseBadge :phase="phase" /></div>
      <el-form label-position="top">
        <el-form-item label="跟进状态">
          <el-radio-group v-model="form.status">
            <el-radio value="pending_followup">待跟进</el-radio>
            <el-radio value="clarified">已澄清</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="疑点内容（最多 1000 字）">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="4"
            maxlength="1000"
            show-word-limit
            placeholder="记录该阶段的异常现象、排查方向或澄清结论；不改动原始评分与输入指纹。"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存说明</el-button>
      </template>
    </el-dialog>
  </div>
</template>
