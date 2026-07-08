<template>
  <Teleport to="body">
    <Transition name="drawer-fade">
      <div
        v-if="modelValue"
        class="conversation-info-mask"
        @click.self="close"
      >
        <Transition name="drawer-slide" appear>
          <aside
            class="conversation-info-drawer"
            role="dialog"
            aria-modal="true"
            aria-label="会话信息"
            tabindex="-1"
            @keydown.esc="close"
          >
            <!-- <header class="drawer-header">
              <h2>会话信息</h2>
              <button type="button" class="drawer-close-btn" aria-label="关闭" @click="close">
                <i class="ri-close-line"></i>
              </button>
            </header> -->

            <section class="drawer-section participants-section">
              <div v-if="isGroup" class="member-toolbar">
                <span>群成员</span>
              </div>
              <div class="participant-list">
                <div
                  v-for="user in displayParticipants"
                  :key="user.id"
                  class="participant-item"
                >
                  <img class="participant-avatar" :src="user.avatar || ''" alt="">
                  <span class="participant-name">{{ user.nickname || user.id }}</span>
                </div>
                <button
                  v-if="isGroup"
                  type="button"
                  class="participant-item member-action-entry"
                  title="邀请成员"
                  aria-label="邀请成员"
                  @click="handleInvite"
                >
                  <span class="member-action-icon">
                    <i class="ri-add-line"></i>
                  </span>
                  <span class="participant-name"></span>
                </button>
                <button
                  v-if="isGroup && canManageMembers"
                  type="button"
                  class="participant-item member-action-entry"
                  title="移除成员"
                  aria-label="移除成员"
                  @click="emit('manage-members')"
                >
                  <span class="member-action-icon is-danger">
                    <i class="ri-subtract-line"></i>
                  </span>
                  <span class="participant-name"></span>
                </button>
              </div>
              <button
                v-if="isGroup && hasMoreParticipants"
                type="button"
                class="load-members-btn"
                :disabled="loadingParticipants"
                @click="emit('load-more-members')"
              >
                {{ loadingParticipants ? '加载中...' : '查看更多成员' }}
              </button>
            </section>

            <section class="drawer-section action-section">
              <button v-if="isGroup" type="button" class="drawer-row action-row group-name-action" @click="handleEditGroupName">
                <span>群聊名称</span>
                <span class="drawer-row-value">{{ groupName }}</span>
                <i class="ri-arrow-right-s-line"></i>
              </button>
              <button type="button" class="drawer-row action-row" @click="emit('search')">
                <span>查找聊天内容</span>
                <i class="ri-arrow-right-s-line"></i>
              </button>
            </section>

            <section class="drawer-section setting-section">
              <div class="drawer-row switch-row">
                <span>置顶聊天</span>
                <button
                  type="button"
                  class="switch-control"
                  :class="{ 'is-on': pinned }"
                  :aria-pressed="pinned"
                  @click="togglePinned"
                >
                  <span></span>
                </button>
              </div>
              <div class="drawer-row switch-row">
                <span>消息免打扰</span>
                <button
                  type="button"
                  class="switch-control"
                  :class="{ 'is-on': muted }"
                  :aria-pressed="muted"
                  @click="toggleMuted"
                >
                  <span></span>
                </button>
              </div>
            </section>

            <section class="drawer-section danger-section">
              <button v-if="isGroup" type="button" class="clear-history-btn" @click="emit('leave')">
                退出群聊
              </button>
            </section>
          </aside>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { UserInfo } from '@/types/user'

const props = defineProps<{
  modelValue: boolean
  participants: UserInfo[]
  currentUserRole?: string
  isGroup?: boolean
  groupId?: string
  groupName?: string
  savingGroupName?: boolean
  pinned?: boolean
  muted?: boolean
  hasMoreParticipants?: boolean
  loadingParticipants?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  search: []
  invite: []
  leave: []
  'load-more-members': []
  'update-group-name': [name: string]
  'update-setting': [settings: { pinned?: boolean; muted?: boolean }]
  'edit-group-name': []
  'manage-members': []
}>()

const pinned = ref(false)
const muted = ref(false)

const displayParticipants = computed(() => props.participants.filter((user) => user.id))
const canManageMembers = computed(() => props.currentUserRole === 'owner' || props.currentUserRole === 'admin')

const close = () => {
  emit('update:modelValue', false)
}

const handleEditGroupName = () => {
  emit('edit-group-name')
}

const handleInvite = () => {
  emit('invite')
}

const togglePinned = () => {
  emit('update-setting', { pinned: !pinned.value })
}

const toggleMuted = () => {
  emit('update-setting', { muted: !muted.value })
}

watch(
  () => props.pinned,
  (value) => {
    pinned.value = Boolean(value)
  },
  { immediate: true },
)

watch(
  () => props.muted,
  (value) => {
    muted.value = Boolean(value)
  },
  { immediate: true },
)

watch(
  () => props.modelValue,
  (visible) => {
    document.body.style.overflow = visible ? 'hidden' : ''
  },
)

onUnmounted(() => {
  document.body.style.overflow = ''
})
</script>

<style scoped>
.conversation-info-mask {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  justify-content: flex-end;
  background: rgba(15, 23, 42, 0);
}

.conversation-info-drawer {
  width: min(360px, 100vw);
  height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  background: #f7f8fb;
  outline: none;
  border-left: 1px solid var(--border-color-light);
}

.drawer-header {
  min-height: 52px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  border-bottom: 1px solid var(--border-color-light);
  background: #fff;
}

.drawer-header h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.3;
}

.drawer-close-btn {
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.drawer-close-btn:hover {
  background: #f0f3f8;
  color: var(--text-color-dark);
}

.drawer-close-btn i {
  font-size: 20px;
  line-height: 1;
}

.drawer-section {
  background: #fff;
  border-bottom: 1px solid var(--border-color-light);
}

.participants-section {
  padding: 18px 16px 16px;
}

.member-toolbar {
  min-height: 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  color: var(--text-color-dark);
  font-size: 13px;
  font-weight: 600;
}

.group-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--text-color-secondary);
  font-size: 12px;
}

.group-meta-row strong {
  min-width: 0;
  color: var(--text-color-dark);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.participant-list {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
}

.participant-item {
  width: 52px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 7px;
}

.participant-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  object-fit: cover;
  background: #edf1f7;
}

.participant-name {
  width: 100%;
  color: var(--text-color-secondary);
  font-size: 12px;
  line-height: 1.25;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-action-entry {
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0;
}

.member-action-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  color: var(--text-color-light);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--text-color-light);
}

.member-action-entry:hover .member-action-icon {
  background: var(--bg-color-light);
}

.member-action-icon i {
  font-size: 20px;
  line-height: 1;
}

.load-members-btn {
  width: 100%;
  margin-top: 14px;
  border: none;
  border-radius: 8px;
  background: #f4f6fa;
  color: var(--text-color-secondary);
  padding: 9px 10px;
  font-size: 13px;
  cursor: pointer;
}

.load-members-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.action-section,
.setting-section,
.danger-section {
  border-bottom: 1px solid var(--border-color-light);
}

.drawer-row {
  width: 100%;
  min-height: 54px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
  border: none;
  background: #fff;
  color: var(--text-color-dark);
  font-size: 13px;
  line-height: 1.35;
  text-align: left;
}

.action-row {
  cursor: pointer;
}

.group-name-action > span:first-child {
  flex-shrink: 0;
}

.drawer-row-value {
  min-width: 0;
  flex: 1;
  color: var(--text-color-secondary);
  font-size: 13px;
  font-weight: 400;
  line-height: 1.35;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action-row i {
  flex-shrink: 0;
  color: #a2a9b6;
  font-size: 22px;
  line-height: 1;
}

.switch-row {
  cursor: default;
}

.switch-row + .switch-row {
  border-top: 1px solid var(--border-color-light);
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

.danger-section {
  padding: 14px 16px;
}

.clear-history-btn {
  width: 100%;
  min-height: 42px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--danger-color);
  font-size: 13px;
  line-height: 1.35;
  cursor: pointer;
}

.clear-history-btn:hover {
  background: rgba(255, 77, 79, 0.08);
}

.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 0.18s ease;
}

.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}

.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 0.22s ease;
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .conversation-info-mask {
    background: rgba(15, 23, 42, 0.18);
  }

  .conversation-info-drawer {
    width: min(88vw, 360px);
  }
}
</style>
