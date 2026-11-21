<script setup lang="ts">
import RecordingTab from './RecordingTab.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ readonly state: { readonly path: string } }>()
</script>

<template>
  <RecordingTab :key="state.path" :state="state" />
</template>
