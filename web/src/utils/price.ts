// 格式化价格显示
export const formatPrice = (price: number | null): string => {
  if (price === null) return '0.00'
  return price.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}


/**
 * 格式化价格为友好的显示方式
 * @param price - 原始价格（单位：元）
 * @returns 格式化后的价格字符串
 */
export const getFriendlyPrice = (price: number): string => {
  if (price >= 10000) {
    // 检查是否可以用 "万" 表示
    if (price >= 10000 && price % 100 === 0) {
      // 使用万表示
      return `${(price / 10000).toFixed(2)}万元`;
    } else {
      // 直接显示完整价格，带千分位
      return `${formatPrice(price)} 元`
    }
  } else {
    // 小于 10000 的直接显示完整价格
    return `${formatPrice(price)} 元`
  }
}