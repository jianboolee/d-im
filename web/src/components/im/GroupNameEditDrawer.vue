<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div
        v-if="modelValue"
        class="group-name-edit-mask"
        @click.self="close"
      >
        <Transition name="modal-scale" appear>
          <div
            class="group-name-edit-dialog"
            role="dialog"
            aria-modal="true"
            aria-label="修改群名"
            @keydown.esc="close"
          >
            <header class="dialog-header">
              <h3>修改群名</h3>
              <button type="button" class="dialog-close-btn" aria-label="关闭" @click="close">
                <i class="ri-close-line"></i>
              </button>
            </header>

            <div class="dialog-body">
              <input
                ref="inputRef"
                v-model="draft"
                type="text"
                class="name-input"
                maxlength="40"
                placeholder="请输入群名称"
                :disabled="saving"
                @keydown.enter="submit"
              >
            </div>

            <footer class="dialog-footer">
              <button type="button" class="btn-cancel" :disabled="saving" @click="close">
                取消
              </button>
              <button
                type="button"
                class="btn-save"
                :disabled="!canSubmit"
                @click="submit"
              >
                {{ saving ? '保存中...' : '保存' }}
              </button>
            </footer>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'

const props = defineProps<{
  modelValue: boolean
  groupName: string
  saving?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [name: string]
}>()

const draft = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const normalizedDraft = computed(() => draft.value.trim())
const normalizedName = computed(() => props.groupName.trim())
const canSubmit = computed(() => (
  !props.saving
  && normalizedDraft.value.length > 0
  && normalizedDraft.value !== normalizedName.value
))

const close = () => {
  emit('update:modelValue', false)
}

const submit = () => {
  if (!canSubmit.value) return
  emit('save', normalizedDraft.value)
}

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      draft.value = props.groupName
    }
    void nextTick(() => {
      if (visible && inputRef.value) {
        inputRef.value.focus()
      }
    })
  },
  { immediate: true },
)
</script>

<style scoped>
.group-name-edit-mask {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.36);
}

.group-name-edit-dialog {
  width: min(calc(100vw - 48px), 360px);
  background: #fff;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 12px 36px rgba(15, 23, 42, 0.16);
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px 0;
}

.dialog-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color-dark);
  margin: 0;
}

.dialog-close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #a2a9b6;
  font-size: 20px;
  cursor: pointer;
}

.dialog-close-btn:hover {
  background: #f4f6fa;
  color: var(--text-color-dark);
}

.dialog-body {
  padding: 14px 16px 0;
}

.name-input {
  width: 100%;
  height: 44px;
  border: 1px solid var(--border-color-light);
  border-radius: 10px;
  padding: 0 12px;
  font-size: 14px;
  color: var(--text-color-dark);
  background: #f7f8fb;
  outline: none;
}

.name-input:focus {
  border-color: #4b86f8;
  background: #fff;
}

.name-input:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.dialog-footer {
  display: flex;
  gap: 10px;
  padding: 16px;
}

.btn-cancel,
.btn-save {
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 10px;
  font-size: 14px;
  cursor: pointer;
}

.btn-cancel {
  background: #f4f6fa;
  color: var(--text-color-secondary);
}

.btn-cancel:hover {
  background: #e8ebf0;
}

.btn-save {
  background: #4b86f8;
  color: #fff;
}

.btn-save:hover {
  background: #3d75e6;
}

.btn-save:disabled {
  opacity: 0.46;
  cursor: not-allowed;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.18s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-scale-enter-active,
.modal-scale-leave-active {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.modal-scale-enter-from,
.modal-scale-leave-to {
  transform: scale(0.92);
  opacity: 0;
}
</style>
