<template>
  <div class="group-name-editor">
    <label class="editor-label" for="group-name-input">群名称</label>
    <div class="editor-control">
      <input
        id="group-name-input"
        v-model="draft"
        type="text"
        maxlength="40"
        placeholder="请输入群名称"
        :disabled="saving"
        @keydown.enter="submit"
      >
      <button
        type="button"
        :disabled="!canSubmit"
        @click="submit"
      >
        {{ saving ? '保存中...' : '保存' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  modelValue: string
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [name: string]
}>()

const draft = ref(props.modelValue)

const normalizedDraft = computed(() => draft.value.trim())
const normalizedValue = computed(() => props.modelValue.trim())
const canSubmit = computed(() => (
  !props.saving
  && normalizedDraft.value.length > 0
  && normalizedDraft.value !== normalizedValue.value
))

const submit = () => {
  if (!canSubmit.value) return
  emit('save', normalizedDraft.value)
}

watch(
  () => props.modelValue,
  (value) => {
    draft.value = value
  },
)
</script>

<style scoped>
.group-name-editor {
  padding: 14px 16px;
  background: #fff;
  border-bottom: 1px solid var(--border-color-light);
}

.editor-label {
  display: block;
  margin-bottom: 8px;
  color: var(--text-color-secondary);
  font-size: 12px;
  line-height: 1.3;
}

.editor-control {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-control input {
  min-width: 0;
  height: 36px;
  flex: 1;
  border: 1px solid var(--border-color-light);
  border-radius: 8px;
  background: #f7f8fb;
  color: var(--text-color-dark);
  padding: 0 10px;
  font-size: 13px;
  outline: none;
}

.editor-control input:focus {
  border-color: #4b86f8;
  background: #fff;
}

.editor-control input:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.editor-control button {
  height: 36px;
  flex-shrink: 0;
  border: none;
  border-radius: 8px;
  background: #4b86f8;
  color: #fff;
  padding: 0 12px;
  font-size: 13px;
  cursor: pointer;
}

.editor-control button:disabled {
  cursor: not-allowed;
  opacity: 0.46;
}
</style>
