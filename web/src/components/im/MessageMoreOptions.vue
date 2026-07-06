<!-- 图片消息上传按钮 -->
<template>
  <button
    class="image-upload-btn"
    type="button"
    aria-label="发送图片"
    @click="openImagePicker"
  >
    <i class="ri-image-line"></i>
  </button>
  <input
    ref="imageInputRef"
    type="file"
    multiple
    accept=".jpg,.jpeg,.png,.gif,.webp,image/jpeg,image/png,image/gif,image/webp"
    class="hidden-input"
    @change="handleImageChange"
  >
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { showToast } from '@/plugins/toast'
import { MessageType } from '@/sdk/im'
import { readImageDimensions, getFileFormat } from '@/utils/file'
import { uploadIMFile } from '@/utils/upload'

const IMAGE_MAX_SIZE = 10 * 1024 * 1024
const IMAGE_MAX_COUNT = 9
const IMAGE_ALLOWED_EXTENSIONS = new Set(['jpg', 'jpeg', 'png', 'gif', 'webp'])

/** 文件上传结果 */
interface UploadResult {
  url: string
  size: number
  width?: number
  height?: number
  format?: string
}

type PendingUploadInfo = UploadResult & { uploading?: boolean }

const emit = defineEmits<{
  'select-file': [file: File, type: MessageType, info: PendingUploadInfo]
  'upload-success': [file: File, type: MessageType, info: UploadResult]
  'upload-error': [file: File, type: MessageType]
}>()

const imageInputRef = ref<HTMLInputElement | null>(null)

function openImagePicker() {
  imageInputRef.value?.click()
}

function assertFileSize(file: File, maxBytes: number, message: string): boolean {
  if (file.size <= maxBytes) {
    return true
  }
  showToast(message)
  return false
}

function assertImageCount(files: File[]): boolean {
  if (files.length <= IMAGE_MAX_COUNT) {
    return true
  }
  showToast(`一次最多选择${IMAGE_MAX_COUNT}张图片`)
  return false
}

function getFileExtension(file: File): string {
  const name = file.name.trim().toLowerCase()
  const ext = name.includes('.') ? name.split('.').pop() : ''
  return ext ?? ''
}

function assertImageExtension(file: File): boolean {
  if (IMAGE_ALLOWED_EXTENSIONS.has(getFileExtension(file))) {
    return true
  }
  showToast('仅支持 JPG、PNG、GIF、WEBP 图片')
  return false
}

async function uploadImage(file: File) {
  const localUrl = URL.createObjectURL(file)
  const format = getFileFormat(file)

  emit('select-file', file, MessageType.Image, {
    url: localUrl,
    size: file.size,
    format,
    width: 0,
    height: 0,
    uploading: true,
  })

  try {
    const uploaded = await uploadIMFile(file)
    const dimensions = await readImageDimensions(file)

    const result: UploadResult = {
      url: uploaded.url,
      size: uploaded.size,
      width: uploaded.width ?? dimensions.width,
      height: uploaded.height ?? dimensions.height,
      format: uploaded.format ?? format,
    }

    emit('upload-success', file, MessageType.Image, result)
  } catch (error) {
    console.error('图片上传失败:', error)
    showToast('上传失败')
    emit('upload-error', file, MessageType.Image)
  }
}

async function handleImageChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''

  if (files.length === 0) return
  if (!assertImageCount(files)) return
  if (!files.every(assertImageExtension)) return
  if (!files.every((file) => assertFileSize(file, IMAGE_MAX_SIZE, '图片大小不能超过10MB'))) {
    return
  }

  await Promise.all(files.map((file) => uploadImage(file)))
}
</script>

<style scoped>
.image-upload-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--text-color-dark);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.image-upload-btn i {
  font-size: 20px;
  line-height: 1;
}

.image-upload-btn:active {
  background: rgba(15, 23, 42, 0.08);
  color: var(--text-color-dark);
}

.hidden-input {
  display: none;
}
</style>
