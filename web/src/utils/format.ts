/**
 * 格式化里程数显示
 * @param kilometers 公里数
 * @returns 格式化后的字符串
 */
export function formatMileage(kilometers: number): string {
  if (!kilometers) return ''
  
  if (kilometers >= 10000) {
    return `${parseFloat((kilometers / 10000).toFixed(1))}万`
  }
  return `${kilometers}`
}

// 格式化年份
export const formatShortYear = (year: number): string => {
  if (!year) return ''
  return `${String(year).slice(-2)}年`
}


// 格式化时间为友好格式
export const formatRelativeTime = (time: string) => {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  // 转换为秒
  const seconds = Math.floor(diff / 1000)
  
  // 小于1分钟
  if (seconds < 60) {
    return '刚刚'
  }
  
  // 小于1小时
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    return `${minutes}分钟前`
  }
  
  // 小于24小时
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return `${hours}小时前`
  }
  
  // 小于30天
  const days = Math.floor(hours / 24)
  if (days < 30) {
    return `${days}天前`
  }
  
  // 小于12个月
  const months = Math.floor(days / 30)
  if (months < 12) {
    return `${months}个月前`
  }
  
  // 大于等于12个月
  const years = Math.floor(months / 12)
  return `${years}年前`
}

// 格式化数字为友好显示
export const formatNumber = (num: number): string => {
  if (num >= 10000) {
    return (num / 10000).toFixed(1) + 'w'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'k'
  }
  return num.toString()
} 