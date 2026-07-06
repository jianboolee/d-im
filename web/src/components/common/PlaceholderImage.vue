<template>
  <svg 
    :width="width" 
    :height="height" 
    :viewBox="`0 0 ${aspect === '1:1' ? '300 300' : '400 300'}`"
    xmlns="http://www.w3.org/2000/svg"
  >
    <rect width="100%" height="100%" :fill="bgColor" />
    <text
      x="50%"
      y="50%"
      dominant-baseline="middle"
      text-anchor="middle"
      fill="#ffffff"
      :font-size="aspect === '1:1' ? '14' : '16'"
    >
      {{ text }}
    </text>
  </svg>
</template>

<script setup lang="ts">
defineProps({
  width: {
    type: String,
    default: '100%'
  },
  height: {
    type: String,
    default: '100%'
  },
  text: {
    type: String,
    default: '暂无图片'
  },
  bgColor: {
    type: String,
    default: '#cccccc'
  },
  aspect: {
    type: String,
    default: '4:3',
    validator: (value: string) => ['4:3', '1:1'].includes(value)
  }
})
</script> 