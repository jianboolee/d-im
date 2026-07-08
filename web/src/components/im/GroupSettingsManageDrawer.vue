<template>
  <Teleport to="body">
    <Transition name="settings-drawer-fade">
      <div
        v-if="modelValue"
        class="settings-drawer-mask"
        @click.self="close"
      >
        <Transition name="settings-drawer-slide" appear>
          <form
            class="settings-drawer"
            role="dialog"
            aria-modal="true"
            aria-label="群管理"
            @submit.prevent="submit"
          >
            <header class="settings-drawer-header">
              <button type="button" class="settings-drawer-back" aria-label="返回" :disabled="loading" @click="close">
                <i class="ri-arrow-left-s-line"></i>
              </button>
              <h2>群管理</h2>
              <span></span>
            </header>

            <section class="settings-drawer-content">
              <label class="settings-row">
                <span>入群方式</span>
                <select v-model="draft.join_method" :disabled="loading">
                  <option value="free">自由加入</option>
                  <option value="verify">需要验证</option>
                  <option value="invite">仅邀请</option>
                  <option value="forbidden">禁止加入</option>
                </select>
              </label>

              <div class="settings-row">
                <span>公开群</span>
                <button
                  type="button"
                  class="switch-control"
                  :class="{ 'is-on': draft.is_public }"
                  :aria-pressed="draft.is_public"
                  :disabled="loading"
                  @click="draft.is_public = !draft.is_public"
                >
                  <span></span>
                </button>
              </div>

              <div class="settings-row">
                <span>全员禁言</span>
                <button
                  type="button"
                  class="switch-control"
                  :class="{ 'is-on': draft.is_muted_all }"
                  :aria-pressed="draft.is_muted_all"
                  :disabled="loading"
                  @click="draft.is_muted_all = !draft.is_muted_all"
                >
                  <span></span>
                </button>
              </div>

              <div class="settings-row">
                <span>成员可邀请</span>
                <button
                  type="button"
                  class="switch-control"
                  :class="{ 'is-on': draft.allow_member_invite }"
                  :aria-pressed="draft.allow_member_invite"
                  :disabled="loading"
                  @click="draft.allow_member_invite = !draft.allow_member_invite"
                >
                  <span></span>
                </button>
              </div>

              <p v-if="error" class="settings-error">{{ error }}</p>
            </section>

            <footer class="settings-drawer-actions">
              <button type="button" class="settings-cancel" :disabled="loading" @click="close">
                取消
              </button>
              <button type="submit" class="settings-submit" :disabled="loading">
                {{ loading ? '保存中' : '保存' }}
              </button>
            </footer>
          </form>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { GroupSettings, GroupSettingsPatch } from '@/sdk/im'

const props = defineProps<{
  modelValue: boolean
  settings?: GroupSettings
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [settings: GroupSettingsPatch]
}>()

const draft = reactive<Required<GroupSettingsPatch>>({
  join_method: 'free',
  is_muted_all: false,
  is_public: true,
  allow_member_invite: true,
})

const syncDraft = () => {
  draft.join_method = props.settings?.join_method ?? 'free'
  draft.is_muted_all = Boolean(props.settings?.is_muted_all)
  draft.is_public = props.settings?.is_public ?? true
  draft.allow_member_invite = props.settings?.allow_member_invite ?? true
}

const close = () => {
  if (props.loading) return
  emit('update:modelValue', false)
}

const submit = () => {
  emit('submit', {
    join_method: draft.join_method,
    is_muted_all: draft.is_muted_all,
    is_public: draft.is_public,
    allow_member_invite: draft.allow_member_invite,
  })
}

watch(
  () => [props.modelValue, props.settings] as const,
  ([visible]) => {
    if (visible) {
      syncDraft()
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.settings-drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  justify-content: flex-end;
  background: rgba(15, 23, 42, 0);
}

.settings-drawer {
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

.settings-drawer-header {
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

.settings-drawer-back {
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

.settings-drawer-back:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.settings-drawer-back:not(:disabled):hover {
  background: #f0f3f8;
  color: var(--text-color-dark);
}

.settings-drawer-back i {
  font-size: 24px;
  line-height: 1;
}

.settings-drawer-header h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 15px;
  font-weight: 600;
  text-align: center;
}

.settings-drawer-content {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  background: #fff;
}

.settings-row {
  min-height: 54px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
  border-bottom: 1px solid var(--border-color-light);
  color: var(--text-color-dark);
  font-size: 13px;
}

.settings-row > span {
  flex-shrink: 0;
}

.settings-row select {
  min-width: 120px;
  border: 1px solid var(--border-color-light);
  border-radius: 8px;
  background: #fff;
  color: var(--text-color-dark);
  padding: 7px 9px;
  font-size: 13px;
  outline: none;
}

.settings-row select:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.switch-control {
  width: 36px;
  height: 22px;
  flex-shrink: 0;
  padding: 2px;
  border: none;
  border-radius: 999px;
  background: #e8ebf0;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.switch-control:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.switch-control span {
  width: 18px;
  height: 18px;
  display: block;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 4px rgba(15, 23, 42, 0.18);
  transition: transform 0.2s ease;
}

.switch-control.is-on {
  background: #07c160;
}

.switch-control.is-on span {
  transform: translateX(14px);
}

.settings-error {
  margin: 12px 16px 0;
  color: var(--danger-color);
  font-size: 13px;
  line-height: 1.4;
}

.settings-drawer-actions {
  flex-shrink: 0;
  display: flex;
  gap: 10px;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--border-color-light);
  background: #fff;
}

.settings-cancel,
.settings-submit {
  min-width: 0;
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.settings-cancel {
  background: #f3f5f9;
  color: var(--text-color-secondary);
}

.settings-submit {
  background: #4b86f8;
  color: #fff;
}

.settings-cancel:disabled,
.settings-submit:disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.settings-drawer-fade-enter-active,
.settings-drawer-fade-leave-active {
  transition: opacity 0.18s ease;
}

.settings-drawer-fade-enter-from,
.settings-drawer-fade-leave-to {
  opacity: 0;
}

.settings-drawer-slide-enter-active,
.settings-drawer-slide-leave-active {
  transition: transform 0.22s ease;
}

.settings-drawer-slide-enter-from,
.settings-drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .settings-drawer-mask {
    background: rgba(15, 23, 42, 0.18);
  }

  .settings-drawer {
    width: min(88vw, 360px);
  }
}
</style>
