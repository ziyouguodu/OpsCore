<script setup>
import { iconPath } from '../ui/icons'

defineProps({
  name: { type: String, required: true }
})
</script>

<template>
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path v-for="path in iconPath(name)" :key="path" :d="path" />
  </svg>
</template>
