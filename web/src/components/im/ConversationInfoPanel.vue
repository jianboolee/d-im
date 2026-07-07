<template>
  <ConversationInfoDrawer
    :model-value="modelValue"
    :participants="displayParticipants"
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
    @leave="showLeaveConfirm = true"
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
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from '@/plugins/toast'
import { useIMStore } from '@/stores/im'
import { useConversationList } from '@/composables/useConversationList'
import { useUserProfiles } from '@/composables/useUserProfiles'
import ConversationInfoDrawer from '@/components/im/ConversationInfoDrawer.vue'
import ConversationMessageSearchModal from '@/components/im/ConversationMessageSearchModal.vue'
import GroupNameEditDrawer from '@/components/im/GroupNameEditDrawer.vue'
import ConfirmModal from '@/components/common/ConfirmModal.vue'
import type { Conversation, GroupMember, UserInfo } from '@/sdk/im'

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
const { updateConversationMemberState, updateConversationGroupInfo, removeConversation } = useConversationList()
const { userMap, mergeUsers } = useUserProfiles()

const groupDetailMembers = ref<GroupMember[]>([])
const groupName = ref('')
const groupMembersNextCursor = ref<string | undefined>()
const groupMembersHasMore = ref(false)
const loadingGroupMembers = ref(false)
const savingGroupName = ref(false)
const showMessageSearch = ref(false)
const showLeaveConfirm = ref(false)
const showGroupNameDrawer = ref(false)
const leavingGroup = ref(false)

const isGroup = computed(() => props.conversation?.chat_type === 'group')
const displayParticipants = computed<UserInfo[]>(() => {
  if (!isGroup.value) return props.participants

  return groupDetailMembers.value
    .map((member) => member.user_info ?? userMap.value[member.user_id] ?? { id: member.user_id })
    .filter((user) => user.id)
})

const resetGroupState = () => {
  groupDetailMembers.value = []
  groupName.value = ''
  groupMembersNextCursor.value = undefined
  groupMembersHasMore.value = false
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
