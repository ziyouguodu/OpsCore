<script setup>
import { computed } from 'vue'
import { viewMeta } from '../navigation'

const props = defineProps({
  view: { type: String, required: true }
})

const meta = computed(() => viewMeta[props.view])
</script>

<template>
  <header class="workspace-page-header">
    <span class="breadcrumb">{{ meta.breadcrumb }}</span>
    <h1>{{ meta.title }}</h1>
    <p>{{ meta.subtitle }}</p>
  </header>
</template>
