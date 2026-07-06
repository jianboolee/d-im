/** 从 input change 事件读取选中的单个文件，并清空 value 以便重复选择同一文件 */
export function takeInputFile(event: Event): File | null {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  input.value = ''
  return file
}

export function getFileFormat(file: File): string {
  const fromMime = file.type.split('/')[1]
  if (fromMime) return fromMime
  const fromName = file.name.split('.').pop()
  return fromName ?? 'unknown'
}

export function readImageDimensions(file: File): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    const objectUrl = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(objectUrl)
      resolve({ width: img.width, height: img.height })
    }
    img.onerror = () => {
      URL.revokeObjectURL(objectUrl)
      reject(new Error('获取图片尺寸失败'))
    }
    img.src = objectUrl
  })
}
