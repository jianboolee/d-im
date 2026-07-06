import { defineAsyncComponent, type Component } from 'vue'
import { MessageType } from '@/sdk/im'
import TextMessage from './TextMessage.vue'
import ImageMessage from './ImageMessage.vue'
import VoiceMessage from './VoiceMessage.vue'
import CardMessage from './CardMessage.vue'
import MessageStatus from './MessageStatus.vue'
import LinkMessage from './LinkMessage.vue'
import SystemEventMessage from './SystemEventMessage.vue'

const AsyncVideoMessage = defineAsyncComponent(() => import('./VideoMessage.vue'))

export {
  TextMessage,
  ImageMessage,
  VoiceMessage,
  CardMessage,
  MessageStatus,
  LinkMessage,
  SystemEventMessage,
}

export const MessageComponents: Partial<Record<MessageType, Component>> = {
  [MessageType.Text]: TextMessage,
  [MessageType.SystemEvent]: SystemEventMessage,
  [MessageType.Image]: ImageMessage,
  [MessageType.Voice]: VoiceMessage,
  [MessageType.Video]: AsyncVideoMessage,
  [MessageType.Card]: CardMessage,
  [MessageType.Link]: LinkMessage,
}
