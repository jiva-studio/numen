<script setup lang="ts">
/**
 * The view a plugin registered for the slot `titleBar`, drawn with the stylesheet of its plugin.
 * The view reads what it draws from its plugin context. Without a view it draws nothing.
 */
import { computed, type Component } from 'vue'
import { getPluginStyle, type DeclaredSlotView } from '@numen/plugin-host'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  view: DeclaredSlotView | null
}>()

/* --------------------------------- State ---------------------------------- */
const style = computed(() => (props.view ? getPluginStyle(props.view.pluginId) : undefined))
</script>

<template>
  <template v-if="view">
    <link v-if="style" rel="stylesheet" :href="style" />
    <component :is="view.registration.component as Component" :key="view.pluginId" />
  </template>
</template>
