<script setup lang="ts">
/** The dialogs the plugins have open, one over another, each drawn by the component its plugin gave. */
import type { OpenDialog } from '../model/createDialogStack'
import PluginDialog from './PluginDialog.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ entries: readonly OpenDialog[] }>()

const emit = defineEmits<{ (e: 'fail', id: number): void }>()
</script>

<template>
  <PluginDialog v-for="one in entries" :key="one.id" :entry="one" @fail="emit('fail', one.id)" />
</template>
