<template>
  <Teleport to="body">
    <Transition name="invite-drawer-fade">
      <div
        v-if="modelValue"
        class="invite-drawer-mask"
        @click.self="close"
      >
        <Transition name="invite-drawer-slide" appear>
          <form
            class="invite-drawer"
            role="dialog"
            aria-modal="true"
            aria-label="邀请成员"
            @submit.prevent="submit"
          >
            <header class="invite-drawer-header">
              <button type="button" class="invite-drawer-back" aria-label="返回" :disabled="loading" @click="close">
                <i class="ri-arrow-left-s-line"></i>
              </button>
              <h2>邀请成员</h2>
              <span></span>
            </header>

            <section class="invite-drawer-content">
              <label class="invite-field">
                <span>成员用户 ID</span>
                <textarea
                  ref="inputRef"
                  v-model="memberIdsText"
                  rows="6"
                  placeholder="输入用户 ID，多个 ID 可用逗号、空格或换行分隔"
                  :disabled="loading"
                  @keydown.esc="close"
                ></textarea>
              </label>
              <p v-if="displayError" class="invite-error">{{ displayError }}</p>
            </section>

            <footer class="invite-drawer-actions">
              <button type="button" class="invite-cancel" :disabled="loading" @click="close">
                取消
              </button>
              <button type="submit" class="invite-submit" :disabled="loading || parsedMemberIds.length === 0">
                {{ loading ? '邀请中' : '邀请' }}
              </button>
            </footer>
          </form>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  loading?: boolean
  error?: string
  currentUserId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [memberUserIds: string[]]
}>()

const inputRef = ref<HTMLTextAreaElement | null>(null)
const memberIdsText = ref('')
const localError = ref('')

const parseUserIds = (value: string) => Array.from(
  new Set(value.split(/[\s,，;；]+/).map((item) => item.trim()).filter(Boolean)),
)

const parsedMemberIds = computed(() => parseUserIds(memberIdsText.value)
  .filter((id) => id !== props.currentUserId))
const displayError = computed(() => localError.value || props.error || '')

const close = () => {
  if (props.loading) return
  emit('update:modelValue', false)
}

const submit = () => {
  localError.value = ''
  if (parsedMemberIds.value.length === 0) {
    localError.value = '至少输入一个成员用户 ID'
    return
  }
  emit('submit', parsedMemberIds.value)
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) {
      memberIdsText.value = ''
      localError.value = ''
      return
    }
    void nextTick(() => {
      inputRef.value?.focus()
    })
  },
)

watch(
  () => memberIdsText.value,
  () => {
    localError.value = ''
  },
)
</script>

<style scoped>
.invite-drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  justify-content: flex-end;
  background: rgba(15, 23, 42, 0);
}

.invite-drawer {
  width: min(360px, 100vw);
  height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: none;
  border-left: 1px solid var(--border-color-light);
  background: #f7f8fb;
  outline: none;
}

.invite-drawer-header {
  min-height: 54px;
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 36px 1fr 36px;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border-bottom: 1px solid var(--border-color-light);
  background: #fff;
}

.invite-drawer-back {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.invite-drawer-back:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.invite-drawer-back:not(:disabled):hover {
  background: #f0f3f8;
  color: var(--text-color-dark);
}

.invite-drawer-back i {
  font-size: 24px;
  line-height: 1;
}

.invite-drawer-header h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 15px;
  font-weight: 600;
  text-align: center;
}

.invite-drawer-content {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  background: #fff;
  padding: 18px 16px;
}

.invite-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.invite-field span {
  color: var(--text-color-secondary);
  font-size: 13px;
}

.invite-field textarea {
  width: 100%;
  min-height: 132px;
  resize: vertical;
  border: 1px solid var(--border-color-light);
  border-radius: 8px;
  padding: 10px 11px;
  color: var(--text-color-dark);
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

.invite-field textarea:focus {
  border-color: #4b86f8;
}

.invite-field textarea:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.invite-error {
  margin: 12px 0 0;
  color: var(--danger-color);
  font-size: 13px;
  line-height: 1.4;
}

.invite-drawer-actions {
  flex-shrink: 0;
  display: flex;
  gap: 10px;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--border-color-light);
  background: #fff;
}

.invite-cancel,
.invite-submit {
  min-width: 0;
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.invite-cancel {
  background: #f3f5f9;
  color: var(--text-color-secondary);
}

.invite-submit {
  background: #4b86f8;
  color: #fff;
}

.invite-cancel:disabled,
.invite-submit:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.invite-drawer-fade-enter-active,
.invite-drawer-fade-leave-active {
  transition: opacity 0.18s ease;
}

.invite-drawer-fade-enter-from,
.invite-drawer-fade-leave-to {
  opacity: 0;
}

.invite-drawer-slide-enter-active,
.invite-drawer-slide-leave-active {
  transition: transform 0.22s ease;
}

.invite-drawer-slide-enter-from,
.invite-drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .invite-drawer-mask {
    background: rgba(15, 23, 42, 0.18);
  }

  .invite-drawer {
    width: min(88vw, 360px);
  }
}
</style>
