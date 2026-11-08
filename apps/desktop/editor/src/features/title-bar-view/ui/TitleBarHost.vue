<script setup lang="ts">
/**
 * The view a plugin registered for the slot `titleBar`, drawn with the stylesheet of its plugin.
 * The view reads what it draws from its plugin context. Without a view it draws nothing.
 */
import { computed, type Component } from 'vue'
import type { DeclaredSlotView } from '@numen/plugin-host'
import { usePluginStyles } from '@/shared/model/usePluginStyles'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  view: DeclaredSlotView | null
}>()

/* --------------------------------- State ---------------------------------- */
const views = computed(() => (props.view ? [props.view] : []))

/* --------------------------------- Hooks ---------------------------------- */
usePluginStyles(() => views.value)
</script>

<template>
  <component :is="view.registration.component as Component" v-if="view" :key="view.pluginId" />
</template>
