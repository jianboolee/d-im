<template>
  <div class="login-page">
    <form class="content" @submit.prevent="handleLogin">
      <i class="ri-chat-3-line icon"></i>
      <h1>即时消息</h1>
      <label class="field">
        <span>ID</span>
        <input
          v-model.trim="userId"
          name="id"
          autocomplete="username"
          placeholder="用户名或邮箱"
        >
      </label>
      <label class="field">
        <span>密码</span>
        <input
          v-model="password"
          name="password"
          type="password"
          autocomplete="current-password"
          placeholder="输入密码"
        >
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="submitting || !userId || !password">
        {{ submitting ? '登录中...' : '登录' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useIMStore } from '@/stores/im'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const imStore = useIMStore()

const userId = ref(typeof route.query.id === 'string' ? route.query.id : '')
const password = ref('')
const submitting = ref(false)
const error = ref('')

async function handleLogin() {
  if (!userId.value || !password.value || submitting.value) return

  submitting.value = true
  error.value = ''

  try {
    await userStore.login(userId.value, password.value)
    await userStore.fetchUser().catch((fetchError) => {
      console.warn('同步用户信息失败:', fetchError)
    })

    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/im/home'
    await router.replace(redirect)
    imStore.initSDK()
  } catch (err) {
    console.error('登录失败:', err)
    error.value = 'ID 或密码不正确'
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: #f5f6fa;
}

.content {
  max-width: 360px;
  width: min(100%, 360px);
  text-align: center;
  color: #333;
}

.icon {
  font-size: 48px;
  color: #4b86f8;
  margin-bottom: 16px;
}

h1 {
  font-size: 22px;
  margin: 0 0 12px;
}

p {
  margin: 0;
  color: #666;
  line-height: 1.6;
}

.field {
  display: block;
  margin-top: 16px;
  text-align: left;
}

.field span {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #666;
}

.field input {
  width: 100%;
  height: 42px;
  box-sizing: border-box;
  border: 1px solid #d8dde8;
  border-radius: 8px;
  padding: 0 12px;
  font-size: 15px;
  background: #fff;
  color: #222;
}

.field input:focus {
  outline: none;
  border-color: #4b86f8;
  box-shadow: 0 0 0 3px rgba(75, 134, 248, 0.14);
}

.error {
  margin-top: 12px;
  color: #c0392b;
  font-size: 13px;
}

button {
  width: 100%;
  height: 42px;
  margin-top: 18px;
  border: 0;
  border-radius: 8px;
  background: #2563eb;
  color: #fff;
  font-size: 15px;
  cursor: pointer;
}

button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
</style>
