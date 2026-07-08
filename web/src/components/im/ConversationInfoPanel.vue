<template>
  <ConversationInfoDrawer
    :model-value="modelValue"
    :participants="displayParticipants"
    :current-user-role="currentUserRole"
    :is-group="isGroup"
    :group-id="conversation?.group_id"
    :group-name="groupName"
    :pinned="conversation?.pinned"
    :muted="conversation?.muted"
    :has-more-participants="groupMembersHasMore"
    :loading-participants="loadingGroupMembers"
    @update:modelValue="emit('update:modelValue', $event)"
    @invite="emit('invite')"
    @search="showMessageSearch = true"
    @load-more-members="loadMoreGroupMembers"
    @edit-group-name="showGroupNameDrawer = true"
    @update-group-name="handleUpdateGroupName"
    @update-setting="handleUpdateConversationSetting"
    @manage-members="openMemberManageDrawer"
    @manage-group="showGroupSettingsDrawer = true"
    @leave="showLeaveConfirm = true"
  />

  <GroupMemberManageDrawer
    v-model="showMemberManageDrawer"
    :members="memberManageItems"
    :current-user-id="currentUserId"
    :current-user-role="currentUserRole"
    :loading="loadingMemberManageMembers"
    @kick-member="requestKickMember"
  />

  <GroupSettingsManageDrawer
    v-model="showGroupSettingsDrawer"
    :settings="groupSettings"
    :loading="savingGroupSettings"
    :error="groupSettingsError"
    @submit="handleUpdateGroupSettings"
  />

  <ConversationMessageSearchModal
    v-model="showMessageSearch"
    :chat-id="conversation?.chat_id || ''"
  />

  <GroupNameEditDrawer
    v-model="showGroupNameDrawer"
    :group-name="groupName"
    :saving="savingGroupName"
    @save="handleUpdateGroupName"
  />

  <ConfirmModal
    v-model="showLeaveConfirm"
    danger
    title="退出群聊"
    message="退出后将不再显示这个群聊，也不会继续接收群消息。确定要退出吗？"
    confirm-text="退出"
    loading-text="退出中..."
    :loading="leavingGroup"
    @confirm="handleLeaveGroup"
  />

  <ConfirmModal
    v-model="showKickConfirm"
    danger
    title="移除成员"
    :message="kickConfirmMessage"
    confirm-text="移除"
    loading-text="移除中..."
    :loading="kickingMember"
    @confirm="handleKickMember"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from '@/plugins/toast'
import { useIMStore } from '@/stores/im'
import { useUserStore } from '@/stores/user'
import { useConversationList } from '@/composables/useConversationList'
import { useUserProfiles } from '@/composables/useUserProfiles'
import ConversationInfoDrawer from '@/components/im/ConversationInfoDrawer.vue'
import GroupMemberManageDrawer from '@/components/im/GroupMemberManageDrawer.vue'
import GroupSettingsManageDrawer from '@/components/im/GroupSettingsManageDrawer.vue'
import ConversationMessageSearchModal from '@/components/im/ConversationMessageSearchModal.vue'
import GroupNameEditDrawer from '@/components/im/GroupNameEditDrawer.vue'
import ConfirmModal from '@/components/common/ConfirmModal.vue'
import type { Conversation, GroupMember, GroupSettings, GroupSettingsPatch, UserInfo } from '@/sdk/im'

const props = defineProps<{
  modelValue: boolean
  conversation: Conversation | null
  participants: UserInfo[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  invite: []
}>()

const router = useRouter()
const imStore = useIMStore()
const userStore = useUserStore()
const { updateConversationMemberState, updateConversationGroupInfo, removeConversation } = useConversationList()
const { userMap, mergeUsers } = useUserProfiles()

const groupDetailMembers = ref<GroupMember[]>([])
const currentGroupMember = ref<GroupMember | null>(null)
const memberManageMembers = ref<GroupMember[]>([])
const groupSettings = ref<GroupSettings>()
const groupName = ref('')
const groupOwnerId = ref('')
const groupAdminIds = ref<string[]>([])
const groupMembersNextCursor = ref<string | undefined>()
const groupMembersHasMore = ref(false)
const loadingGroupMembers = ref(false)
const savingGroupName = ref(false)
const showMessageSearch = ref(false)
const showLeaveConfirm = ref(false)
const showGroupNameDrawer = ref(false)
const showMemberManageDrawer = ref(false)
const showGroupSettingsDrawer = ref(false)
const showKickConfirm = ref(false)
const leavingGroup = ref(false)
const kickingMember = ref(false)
const loadingMemberManageMembers = ref(false)
const savingGroupSettings = ref(false)
const groupSettingsError = ref('')
const kickTarget = ref<GroupMember | null>(null)

const isGroup = computed(() => props.conversation?.chat_type === 'group')
const currentUserId = computed(() => userStore.userInfo?.id ?? '')
const currentUserRole = computed(() => {
  if (!currentUserId.value) return ''
  if (currentGroupMember.value?.user_id === currentUserId.value) {
    return currentGroupMember.value.role
  }
  const memberRole = groupDetailMembers.value.find((member) => member.user_id === currentUserId.value)?.role
  if (memberRole) return memberRole
  if (groupOwnerId.value === currentUserId.value) return 'owner'
  if (groupAdminIds.value.includes(currentUserId.value)) return 'admin'
  return ''
})
const displayParticipants = computed<UserInfo[]>(() => {
  if (!isGroup.value) return props.participants

  return groupDetailMembers.value
    .map((member) => member.user_info ?? userMap.value[member.user_id] ?? { id: member.user_id })
    .filter((user) => user.id)
})
const memberManageItems = computed(() => {
  if (!isGroup.value) return []
  return memberManageMembers.value.map((member) => {
    const user = member.user_info ?? userMap.value[member.user_id] ?? { id: member.user_id }
    return {
      id: member.id,
      user_id: member.user_id,
      role: member.role,
      user,
    }
  })
})
const kickTargetName = computed(() => {
  if (!kickTarget.value) return ''
  const user = kickTarget.value.user_info ?? userMap.value[kickTarget.value.user_id]
  return user?.nickname || kickTarget.value.group_nickname || '该成员'
})
const kickConfirmMessage = computed(() => `确定将 ${kickTargetName.value} 移出群聊？`)

const resetGroupState = () => {
  groupDetailMembers.value = []
  currentGroupMember.value = null
  memberManageMembers.value = []
  groupSettings.value = undefined
  groupName.value = ''
  groupOwnerId.value = ''
  groupAdminIds.value = []
  groupMembersNextCursor.value = undefined
  groupMembersHasMore.value = false
  kickTarget.value = null
  showMemberManageDrawer.value = false
  showGroupSettingsDrawer.value = false
  showKickConfirm.value = false
  groupSettingsError.value = ''
}

const loadInitialGroupMembers = async () => {
  if (!props.conversation?.group_id || !imStore.imSDK || !isGroup.value) {
    resetGroupState()
    return
  }

  groupName.value = props.conversation.group_info?.name ?? props.conversation.display_name ?? groupName.value
  try {
    loadingGroupMembers.value = true
    const detail = await imStore.imSDK.getGroup(props.conversation.group_id)
    groupName.value = detail.group?.name ?? props.conversation.group_info?.name ?? props.conversation.display_name ?? ''
    groupOwnerId.value = detail.group?.owner_id ?? ''
    groupAdminIds.value = detail.group?.admins ?? []
    groupSettings.value = detail.group?.settings
    currentGroupMember.value = detail.current_member ?? null
    groupDetailMembers.value = detail.members ?? []
    groupMembersNextCursor.value = groupDetailMembers.value[groupDetailMembers.value.length - 1]?.id
    groupMembersHasMore.value = groupDetailMembers.value.length < (detail.group?.member_count ?? 0)
    mergeUsers(groupDetailMembers.value.map((member) => member.user_info))
  } catch (error) {
    console.error('获取群信息失败:', error)
    showToast('群信息加载失败')
  } finally {
    loadingGroupMembers.value = false
  }
}

const loadMoreGroupMembers = async () => {
  const groupId = props.conversation?.group_id
  if (!groupId || !imStore.imSDK || loadingGroupMembers.value || !groupMembersHasMore.value) return

  try {
    loadingGroupMembers.value = true
    const page = await imStore.imSDK.getGroupMembers(groupId, {
      limit: 100,
      cursor: groupMembersNextCursor.value,
    })
    groupDetailMembers.value = [...groupDetailMembers.value, ...(page.items ?? [])]
    groupMembersNextCursor.value = page.next_cursor
    groupMembersHasMore.value = page.has_more
    mergeUsers(page.items.map((member) => member.user_info))
  } catch (error) {
    console.error('加载群成员失败:', error)
    showToast('成员加载失败')
  } finally {
    loadingGroupMembers.value = false
  }
}

const dedupeMembers = (members: GroupMember[]) => {
  const seen = new Set<string>()
  const result: GroupMember[] = []
  for (const member of members) {
    if (!member.user_id || seen.has(member.user_id)) continue
    seen.add(member.user_id)
    result.push(member)
  }
  return result
}

const loadAllGroupMembersForManagement = async () => {
  const groupId = props.conversation?.group_id
  if (!groupId || !imStore.imSDK || loadingMemberManageMembers.value) return

  try {
    loadingMemberManageMembers.value = true
    let cursor: string | undefined
    const members: GroupMember[] = []
    do {
      const page = await imStore.imSDK.getGroupMembers(groupId, {
        limit: 100,
        cursor,
      })
      members.push(...(page.items ?? []))
      cursor = page.has_more ? page.next_cursor : undefined
    } while (cursor)
    memberManageMembers.value = dedupeMembers(members)
    mergeUsers(memberManageMembers.value.map((member) => member.user_info))
  } catch (error) {
    console.error('加载成员管理列表失败:', error)
    showToast('成员加载失败')
  } finally {
    loadingMemberManageMembers.value = false
  }
}

const openMemberManageDrawer = () => {
  showMemberManageDrawer.value = true
  void loadAllGroupMembersForManagement()
}

const handleUpdateGroupName = async (name: string) => {
  const groupId = props.conversation?.group_id
  const conversationId = props.conversation?.id
  if (!groupId || !conversationId || !imStore.imSDK || savingGroupName.value) return

  const previousName = groupName.value
  try {
    savingGroupName.value = true
    groupName.value = name
    const detail = await imStore.imSDK.updateGroup(groupId, { name })
    if (detail.group) {
      groupName.value = detail.group.name
      groupOwnerId.value = detail.group.owner_id
      groupAdminIds.value = detail.group.admins ?? groupAdminIds.value
      updateConversationGroupInfo(conversationId, {
        id: detail.group.id,
        name: detail.group.name,
        avatar_url: detail.group.avatar_url,
        member_count: detail.group.member_count,
      })
    }
    showGroupNameDrawer.value = false
    showToast('群名称已更新')
  } catch (error) {
    console.error('修改群名失败:', error)
    groupName.value = previousName
    showToast('修改失败')
  } finally {
    savingGroupName.value = false
  }
}

const handleUpdateConversationSetting = async (settings: { pinned?: boolean; muted?: boolean }) => {
  const conversation = props.conversation
  if (!imStore.imSDK || !conversation?.id) return

  const previous = {
    pinned: conversation.pinned,
    muted: conversation.muted,
  }
  const optimistic = { ...previous, ...settings }
  updateConversationMemberState(conversation.id, optimistic)

  try {
    const updated = await imStore.imSDK.updateConversationSettings(conversation.id, settings)
    updateConversationMemberState(conversation.id, updated)
  } catch (error) {
    console.error('更新会话设置失败:', error)
    updateConversationMemberState(conversation.id, previous)
    showToast('设置失败')
  }
}

const handleUpdateGroupSettings = async (settings: GroupSettingsPatch) => {
  const groupId = props.conversation?.group_id
  const conversationId = props.conversation?.id
  if (!groupId || !conversationId || !imStore.imSDK || savingGroupSettings.value) return

  try {
    savingGroupSettings.value = true
    groupSettingsError.value = ''
    const detail = await imStore.imSDK.updateGroupSettings(groupId, settings)
    if (detail.group) {
      groupSettings.value = detail.group.settings
      groupOwnerId.value = detail.group.owner_id
      groupAdminIds.value = detail.group.admins ?? groupAdminIds.value
      updateConversationGroupInfo(conversationId, {
        id: detail.group.id,
        name: detail.group.name,
        avatar_url: detail.group.avatar_url,
        member_count: detail.group.member_count,
      })
    }
    showGroupSettingsDrawer.value = false
    showToast('群管理已更新')
  } catch (error) {
    console.error('更新群设置失败:', error)
    groupSettingsError.value = error instanceof Error ? error.message : '更新失败'
  } finally {
    savingGroupSettings.value = false
  }
}

const requestKickMember = (userId: string) => {
  const target = memberManageMembers.value.find((member) => member.user_id === userId)
    ?? groupDetailMembers.value.find((member) => member.user_id === userId)
  if (!target || kickingMember.value) return

  kickTarget.value = target
  showKickConfirm.value = true
}

const handleKickMember = async () => {
  const groupId = props.conversation?.group_id
  const conversationId = props.conversation?.id
  const target = kickTarget.value
  if (!imStore.imSDK || !groupId || !conversationId || !target || kickingMember.value) return

  try {
    kickingMember.value = true
    const detail = await imStore.imSDK.kickGroupMember(groupId, target.user_id)
    groupDetailMembers.value = groupDetailMembers.value.filter((member) => member.user_id !== target.user_id)
    memberManageMembers.value = memberManageMembers.value.filter((member) => member.user_id !== target.user_id)
    if (detail.group) {
      groupOwnerId.value = detail.group.owner_id
      groupAdminIds.value = detail.group.admins ?? groupAdminIds.value
      updateConversationGroupInfo(conversationId, {
        id: detail.group.id,
        name: detail.group.name,
        avatar_url: detail.group.avatar_url,
        member_count: detail.group.member_count,
      })
      groupMembersHasMore.value = groupDetailMembers.value.length < detail.group.member_count
    }
    showKickConfirm.value = false
    kickTarget.value = null
    showToast('已移除成员')
  } catch (error) {
    console.error('移除群成员失败:', error)
    showToast('移除失败')
  } finally {
    kickingMember.value = false
  }
}

const handleLeaveGroup = async () => {
  const groupId = props.conversation?.group_id
  const conversationId = props.conversation?.id
  if (!imStore.imSDK || !groupId || !conversationId || leavingGroup.value) return

  try {
    leavingGroup.value = true
    await imStore.imSDK.leaveGroup(groupId)
    removeConversation(conversationId)
    showLeaveConfirm.value = false
    emit('update:modelValue', false)
    router.replace({ name: 'im-home' })
  } catch (error) {
    console.error('退出群聊失败:', error)
    showToast('退出失败')
  } finally {
    leavingGroup.value = false
  }
}

watch(
  () => [props.modelValue, props.conversation?.id, props.conversation?.group_id] as const,
  ([visible]) => {
    if (!visible) {
      showMessageSearch.value = false
      showLeaveConfirm.value = false
      showMemberManageDrawer.value = false
      showGroupSettingsDrawer.value = false
      showKickConfirm.value = false
      resetGroupState()
      return
    }

    if (isGroup.value) {
      void loadInitialGroupMembers()
      return
    }

    resetGroupState()
  },
  { immediate: true },
)
</script>
