<template>
  <Teleport to="body">
    <Transition name="member-drawer-fade">
      <div
        v-if="modelValue"
        class="member-manage-mask"
        @click.self="close"
      >
        <Transition name="member-drawer-slide" appear>
          <aside
            class="member-manage-drawer"
            role="dialog"
            aria-modal="true"
            aria-label="成员管理"
            tabindex="-1"
            @keydown.esc="close"
          >
            <header class="member-manage-header">
              <button type="button" class="member-manage-back" aria-label="返回" @click="close">
                <i class="ri-arrow-left-s-line"></i>
              </button>
              <h2>成员管理</h2>
              <span class="member-manage-count">{{ members.length }}</span>
            </header>

            <section class="member-manage-content">
              <div v-if="loading" class="member-manage-loading">加载中...</div>
              <div v-else class="member-manage-list">
                <div
                  v-for="member in members"
                  :key="member.user_id"
                  class="member-manage-item"
                >
                  <img class="member-manage-avatar" :src="member.user.avatar || ''" alt="">
                  <div class="member-manage-body">
                    <div class="member-manage-name-row">
                      <span class="member-manage-name">{{ member.user.nickname || member.user.id }}</span>
                      <span v-if="roleLabel(member.role)" class="member-role-tag">{{ roleLabel(member.role) }}</span>
                    </div>
                  </div>
                  <button
                    v-if="canKickMember(member)"
                    type="button"
                    class="member-kick-btn"
                    title="移除成员"
                    aria-label="移除成员"
                    @click="emit('kick-member', member.user_id)"
                  >
                    <i class="ri-delete-bin-line"></i>
                  </button>
                </div>
              </div>
            </section>
          </aside>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import type { UserInfo } from '@/types/user'

const props = defineProps<{
  modelValue: boolean
  members: GroupMemberItem[]
  currentUserId?: string
  currentUserRole?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'kick-member': [userId: string]
}>()

export interface GroupMemberItem {
  id: string
  user_id: string
  role: string
  user: UserInfo
}

const close = () => {
  emit('update:modelValue', false)
}

const roleLabel = (role: string) => {
  if (role === 'owner') return '群主'
  if (role === 'admin') return '管理员'
  return ''
}

const canKickMember = (member: GroupMemberItem) => {
  const currentUserId = props.currentUserId
  const currentUserRole = props.currentUserRole
  if (!currentUserId || member.user_id === currentUserId) return false
  if (currentUserRole === 'owner') return member.role !== 'owner'
  if (currentUserRole === 'admin') return member.role === 'member'
  return false
}
</script>

<style scoped>
.member-manage-mask {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  justify-content: flex-end;
  background: rgba(15, 23, 42, 0);
}

.member-manage-drawer {
  width: min(360px, 100vw);
  height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-left: 1px solid var(--border-color-light);
  background: #f7f8fb;
  outline: none;
}

.member-manage-header {
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

.member-manage-back {
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

.member-manage-back:hover {
  background: #f0f3f8;
  color: var(--text-color-dark);
}

.member-manage-back i {
  font-size: 24px;
  line-height: 1;
}

.member-manage-header h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 15px;
  font-weight: 600;
  text-align: center;
}

.member-manage-count {
  color: var(--text-color-secondary);
  font-size: 12px;
  text-align: right;
}

.member-manage-content {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  background: #fff;
  padding: 8px 16px 16px;
}

.member-manage-loading {
  padding: 24px 0;
  color: var(--text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.member-manage-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.member-manage-item {
  min-height: 52px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-radius: 8px;
}

.member-manage-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  border-radius: 8px;
  object-fit: cover;
  background: #edf1f7;
}

.member-manage-body {
  min-width: 0;
  flex: 1;
}

.member-manage-name-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.member-manage-name {
  min-width: 0;
  color: var(--text-color-dark);
  font-size: 13px;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-role-tag {
  flex-shrink: 0;
  border-radius: 5px;
  background: #eef4ff;
  color: #4b86f8;
  padding: 2px 5px;
  font-size: 11px;
  line-height: 1.2;
}

.member-kick-btn {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--danger-color);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.member-kick-btn:hover {
  background: rgba(255, 77, 79, 0.08);
}

.member-kick-btn i {
  font-size: 17px;
  line-height: 1;
}

.member-drawer-fade-enter-active,
.member-drawer-fade-leave-active {
  transition: opacity 0.18s ease;
}

.member-drawer-fade-enter-from,
.member-drawer-fade-leave-to {
  opacity: 0;
}

.member-drawer-slide-enter-active,
.member-drawer-slide-leave-active {
  transition: transform 0.22s ease;
}

.member-drawer-slide-enter-from,
.member-drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .member-manage-mask {
    background: rgba(15, 23, 42, 0.18);
  }

  .member-manage-drawer {
    width: min(88vw, 360px);
  }
}
</style>
