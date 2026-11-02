<script setup lang="ts">
/**
 * The page of a plugin the catalog offers and this installation does not have. The application
 * has no installer, so the page describes the plugin and offers no action.
 */
import { PluginDetail as PluginDetailView, type ManagedPlugin } from '@numen/plugin-manager'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  plugin: ManagedPlugin
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <PluginDetailView
    :plugin="props.plugin"
    :is-enabled="false"
    readme=""
    :data-file-count="0"
    problem=""
    :can-install="false"
    :has-settings="false"
    @back="onBack"
  />
</template>
