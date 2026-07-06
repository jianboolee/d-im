<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleConfirm">
      <div class="modal-card">
        <div class="modal-icon">
          <i class="ri-time-line"></i>
        </div>
        <h2 class="modal-title">会话已过期</h2>
        <p class="modal-desc">
          您的登录会话已过期，请从业务系统重新进入聊天。
        </p>
        <button type="button" class="modal-btn" @click="handleConfirm">
          知道了
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'confirm'): void
}>()

const handleConfirm = () => {
  emit('confirm')
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(2px);
  animation: fade-in 0.2s ease;
}

.modal-card {
  width: 320px;
  max-width: calc(100vw - 48px);
  padding: 32px 24px 24px;
  border-radius: 16px;
  background: #fff;
  text-align: center;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.18);
  animation: scale-in 0.25s ease;
}

.modal-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  border-radius: 50%;
  background: #fef3c7;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-icon i {
  font-size: 28px;
  color: #d97706;
  line-height: 1;
}

.modal-title {
  font-size: 17px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px;
}

.modal-desc {
  font-size: 14px;
  color: #64748b;
  line-height: 1.6;
  margin: 0 0 20px;
}

.modal-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 120px;
  height: 40px;
  padding: 0 20px;
  border: none;
  border-radius: 10px;
  background: #4b86f8;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.modal-btn:hover {
  background: #3b6fde;
}

.modal-btn:active {
  background: #2f5ec7;
}

@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes scale-in {
  from { opacity: 0; transform: scale(0.92); }
  to { opacity: 1; transform: scale(1); }
}
</style>
