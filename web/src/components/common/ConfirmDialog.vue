<template>
  <Teleport to="body">
    <div v-if="visible" class="confirm-overlay" @click.self="handleCancel">
      <div class="confirm-card">
        <h3 class="confirm-title">{{ title }}</h3>
        <p v-if="message" class="confirm-message">{{ message }}</p>
        <p v-if="detail" class="confirm-detail">{{ detail }}</p>
        <div class="confirm-actions">
          <button type="button" class="confirm-btn confirm-btn-cancel" @click="handleCancel">
            {{ cancelText }}
          </button>
          <button type="button" class="confirm-btn confirm-btn-primary" @click="handleConfirm">
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
  title: string
  message?: string
  detail?: string
  confirmText?: string
  cancelText?: string
}>()

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const handleConfirm = () => emit('confirm')
const handleCancel = () => emit('cancel')
</script>

<style scoped>
.confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(2px);
  animation: confirm-fade-in 0.2s ease;
}

.confirm-card {
  width: 300px;
  max-width: calc(100vw - 48px);
  padding: 28px 24px 20px;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.18);
  animation: confirm-scale-in 0.25s ease;
}

.confirm-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px;
  line-height: 1.4;
}

.confirm-message {
  font-size: 14px;
  color: #475569;
  line-height: 1.6;
  margin: 0 0 4px;
}

.confirm-detail {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
  margin: 4px 0 0;
  word-break: break-all;
}

.confirm-actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}

.confirm-btn {
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.confirm-btn-cancel {
  background: #f1f5f9;
  color: #475569;
}

.confirm-btn-cancel:hover {
  background: #e2e8f0;
}

.confirm-btn-primary {
  background: #4b86f8;
  color: #fff;
}

.confirm-btn-primary:hover {
  background: #3b6fde;
}

.confirm-btn-primary:active {
  background: #2f5ec7;
}

@keyframes confirm-fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes confirm-scale-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}
</style>
