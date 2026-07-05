<template>
  <div class="login-page">
    <div class="content">
      <h1>即时消息</h1>
      <p class="desc">开发模式登录</p>

      <div class="form">
        <input
          v-model="uid"
          type="text"
          placeholder="用户 ID（如 user_001）"
          class="input"
          autofocus
        />
        <input
          v-model="deviceId"
          type="text"
          placeholder="设备 ID（如 web_chrome_v1）"
          class="input"
        />
        <button :disabled="!uid || loggingIn" class="btn" @click="handleLogin">
          {{ loggingIn ? '登录中…' : '进入聊天' }}
        </button>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const uid = ref('')
const deviceId = ref('web_chrome_v1')
const loggingIn = ref(false)
const error = ref('')

async function handleLogin() {
  if (!uid.value) return
  loggingIn.value = true
  error.value = ''

  try {
    await userStore.login(uid.value, deviceId.value)
    router.replace({ name: 'im-chat' })
  } catch (e: any) {
    error.value = e?.message || '登录失败'
  } finally {
    loggingIn.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f6fa;
}

.content {
  width: 360px;
  text-align: center;
}

h1 {
  font-size: 24px;
  margin: 0 0 4px;
  color: #333;
}

.desc {
  color: #999;
  margin: 0 0 24px;
  font-size: 14px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.input {
  padding: 12px 16px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 15px;
  outline: none;
}

.input:focus {
  border-color: #4b86f8;
}

.btn {
  padding: 12px;
  background: #4b86f8;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error {
  color: #c0392b;
  margin: 0;
  font-size: 14px;
}
</style>