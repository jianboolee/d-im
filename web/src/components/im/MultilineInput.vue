<template>
  <div class="multiline-input">
    <!-- honeypot: 骗走 Chrome 自动填充，避免填进聊天输入框 -->
    <input
      type="text"
      autocomplete="off"
      readonly
      tabindex="-1"
      aria-hidden="true"
      class="autofill-honeypot"
    />
    <textarea
      ref="textareaRef"
      :value="modelValue"
      @input="handleInput"
      @compositionstart="handleCompositionStart"
      @compositionend="handleCompositionEnd"
      @keydown.enter="handleEnter"
      @focus="handleFocus"
      @blur="handleBlur"
      :placeholder="placeholder"
      :rows="minRows"
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
      spellcheck="false"
      data-lpignore="true"
      data-form-type="other"
    ></textarea>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, onMounted, watch } from 'vue'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  minRows?: number
  maxRows?: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'enter'): void
  (e: 'focus'): void
  (e: 'blur'): void
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const lineHeight = 24
const verticalPadding = 16
const minRows = computed(() => props.minRows || 1)
const maxRows = computed(() => props.maxRows || 4)
const minHeight = computed(() => lineHeight * minRows.value + verticalPadding)
const maxHeight = computed(() => lineHeight * maxRows.value + verticalPadding)
const minHeightCss = computed(() => `${minHeight.value}px`)
const maxHeightCss = computed(() => `${maxHeight.value}px`)
/** IME 组合输入中（如中文拼音选词），回车用于上屏而非发送 */
const isComposing = ref(false)

const handleCompositionStart = () => {
  isComposing.value = true
}

const handleCompositionEnd = () => {
  isComposing.value = false
}

const isImeEnter = (e: KeyboardEvent) => {
  return isComposing.value || e.isComposing || e.keyCode === 229
}

// 调整文本框高度
const adjustHeight = (value = textareaRef.value?.value ?? props.modelValue) => {
  const textarea = textareaRef.value
  if (!textarea) return

  // 如果内容为空，重置为初始高度
  if (value.length === 0) {
    textarea.style.height = `${minHeight.value}px`
    textarea.style.overflowY = 'hidden'
    return
  }
  
  // 重置高度以获取实际滚动高度
  textarea.style.height = 'auto'
  
  // 计算新高度
  const scrollHeight = textarea.scrollHeight
  // 设置新高度，不超过最大高度
  const nextHeight = Math.min(Math.max(scrollHeight, minHeight.value), maxHeight.value)
  textarea.style.height = `${nextHeight}px`
  textarea.style.overflowY = scrollHeight > maxHeight.value ? 'auto' : 'hidden'
  if (scrollHeight > maxHeight.value) {
    textarea.scrollTop = textarea.scrollHeight
  }
}

// 处理输入
const handleInput = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
  adjustHeight(target.value)
}

const handleFocus = () => {
  emit('focus')
}

const handleBlur = () => {
  if (!props.modelValue.trim()) {
    adjustHeight('')
  }
  emit('blur')
}

const insertNewline = () => {
  const textarea = textareaRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const currentValue = textarea.value
  const nextValue = `${currentValue.slice(0, start)}\n${currentValue.slice(end)}`

  emit('update:modelValue', nextValue)
  requestAnimationFrame(() => {
    textarea.selectionStart = start + 1
    textarea.selectionEnd = start + 1
    adjustHeight()
  })
}

// 处理回车键：IME 组合中不发送；Shift/Ctrl/Command+Enter 换行；Enter 发送
const handleEnter = (e: KeyboardEvent) => {
  if (isImeEnter(e)) {
    return
  }

  if (e.shiftKey || e.ctrlKey || e.metaKey) {
    e.preventDefault()
    insertNewline()
    return
  }

  e.preventDefault()
  emit('enter')
}

// 监听值变化
watch(() => props.modelValue, async () => {
  await nextTick()
  adjustHeight()
})

watch([minRows, maxRows], async () => {
  await nextTick()
  adjustHeight()
})

onMounted(() => {
  adjustHeight()
})
</script>

<style scoped>
/* honeypot: 隐藏的输入框，专门骗走 Chrome 自动填充 */
.autofill-honeypot {
  position: absolute;
  opacity: 0;
  height: 0;
  width: 0;
  pointer-events: none;
}

.multiline-input {
  width: 100%;
  min-width: 0;
  position: relative;
  max-height: v-bind(maxHeightCss);
}

.multiline-input textarea {
  width: 100%;
  resize: none;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  line-height: 24px;
  padding: 8px 0;
  box-sizing: border-box;
  overflow-y: hidden;
  display: block;
  min-height: v-bind(minHeightCss);
}

/* 自定义滚动条样式 */
.multiline-input textarea::-webkit-scrollbar {
  width: 4px;
}

.multiline-input textarea::-webkit-scrollbar-track {
  background: transparent;
}

.multiline-input textarea::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 2px;
}
</style> 
