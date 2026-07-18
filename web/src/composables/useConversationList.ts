import { storeToRefs } from 'pinia'
import { useConversationListStore } from '@/stores/conversationList'

export function useConversationList() {
  const store = useConversationListStore()

  return {
    ...storeToRefs(store),
    loadConversations: store.loadConversations,
    loadMoreConversations: store.loadMoreConversations,
    searchConversations: store.searchConversations,
    loadMoreSearchConversations: store.loadMoreSearchConversations,
    handleIncomingMessage: store.handleIncomingMessage,
    clearUnreadForConversation: store.clearUnreadForConversation,
    upsertConversation: store.upsertConversation,
    updateConversationMemberState: store.updateConversationMemberState,
    updateConversationGroupInfo: store.updateConversationGroupInfo,
    updateConversationGroupInfoByGroupId: store.updateConversationGroupInfoByGroupId,
    removeConversation: store.removeConversation,
    ensureConversationInList: store.ensureConversationInList,
    ensureConversationByChatId: store.ensureConversationByChatId,
    requestScrollToConversation: store.requestScrollToConversation,
    getPeerUserIds: store.getPeerUserIds,
    resetConversations: store.resetConversations,
  }
}
