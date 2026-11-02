<script setup lang="ts">
/**
 * The components plugins registered for the status bar, drawn in the order they were registered,
 * with the stylesheet of each plugin that registered one.
 */
import type { DeclaredStatusView } from '@numen/plugin-host'
import { usePluginStyles } from '../model/usePluginStyles'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  views: readonly DeclaredStatusView[]
}>()

/* --------------------------------- Hooks ---------------------------------- */
usePluginStyles(() => props.views)
</script>

<template>
  <div class="status-views">
    <component
      :is="view.registration.component"
      v-for="view in views"
      :key="`${view.pluginId}:${view.registration.id}`"
      :plugin-id="view.pluginId"
    />
  </div>
</template>

<style scoped>
.status-views {
  display: inline-flex;
  align-items: center;
  block-size: 100%;
}
</style>
