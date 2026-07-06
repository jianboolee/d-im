<template>
  <Teleport to="body">
    <Transition name="confirm-fade">
      <div v-if="modelValue" class="confirm-mask" @click.self="close">
        <div class="confirm-panel" role="dialog" aria-modal="true" :aria-label="title">
          <div class="confirm-icon" :class="{ 'is-danger': danger }">
            <i :class="danger ? 'ri-error-warning-line' : 'ri-information-line'"></i>
          </div>
          <h2>{{ title }}</h2>
          <p>{{ message }}</p>
          <div class="confirm-actions">
            <button type="button" class="confirm-cancel" :disabled="loading" @click="close">
              {{ cancelText }}
            </button>
            <button
              type="button"
              class="confirm-submit"
              :class="{ 'is-danger': danger }"
              :disabled="loading"
              @click="emit('confirm')"
            >
              {{ loading ? loadingText : confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    message: string
    confirmText?: string
    cancelText?: string
    loadingText?: string
    danger?: boolean
    loading?: boolean
  }>(),
  {
    confirmText: '确定',
    cancelText: '取消',
    loadingText: '处理中...',
    danger: false,
    loading: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
}>()

const close = () => {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.confirm-mask {
  position: fixed;
  inset: 0;
  z-index: 1300;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(15, 23, 42, 0.34);
}

.confirm-panel {
  width: min(360px, 100%);
  border-radius: 14px;
  background: #fff;
  padding: 24px 22px 18px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.18);
  text-align: center;
}

.confirm-icon {
  width: 42px;
  height: 42px;
  margin: 0 auto 14px;
  border-radius: 50%;
  background: #edf4ff;
  color: #4b86f8;
  display: flex;
  align-items: center;
  justify-content: center;
}

.confirm-icon.is-danger {
  background: #fff1f0;
  color: var(--danger-color);
}

.confirm-icon i {
  font-size: 22px;
  line-height: 1;
}

.confirm-panel h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 17px;
  font-weight: 700;
}

.confirm-panel p {
  margin: 10px 0 0;
  color: var(--text-color-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.confirm-actions {
  display: flex;
  gap: 10px;
  margin-top: 22px;
}

.confirm-actions button {
  min-width: 0;
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 9px;
  font-size: 14px;
  cursor: pointer;
}

.confirm-actions button:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.confirm-cancel {
  background: #f3f5f9;
  color: var(--text-color-secondary);
}

.confirm-submit {
  background: #4b86f8;
  color: #fff;
}

.confirm-submit.is-danger {
  background: var(--danger-color);
}

.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.18s ease;
}

.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
</style>
