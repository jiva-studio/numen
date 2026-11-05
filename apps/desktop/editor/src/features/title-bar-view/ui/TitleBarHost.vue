<script setup lang="ts">
/**
 * The view a plugin registered for the slot `titleBar`, drawn with the stylesheet of its plugin and
 * given what the window knows of the bar. Without a view it draws nothing.
 */
import { computed, type Component } from 'vue'
import type { DeclaredSlotView, TitleBarViewProps } from '@numen/plugin-host'
import { usePluginStyles } from '@/shared/model/usePluginStyles'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<
  Omit<TitleBarViewProps, 'pluginId'> & {
    /** The view of the slot, or null while no plugin draws the bar. */
    view: DeclaredSlotView | null
  }
>()

const emit = defineEmits<{
  (event: 'run', id: string): void
  (event: 'choose', id: string): void
  (event: 'minimise'): void
  (event: 'toggle-maximise'): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const views = computed(() => (props.view ? [props.view] : []))

/* --------------------------------- Hooks ---------------------------------- */
usePluginStyles(() => views.value)
</script>

<template>
  <component
    :is="view.registration.component as Component"
    v-if="view"
    :key="view.pluginId"
    :plugin-id="view.pluginId"
    :app-name="appName"
    :vault-name="vaultName"
    :controls="controls"
    :is-maximised="isMaximised"
    :is-fullscreen="isFullscreen"
    :actions="actions"
    :choices="choices"
    @run="(id: string) => emit('run', id)"
    @choose="(id: string) => emit('choose', id)"
    @minimise="emit('minimise')"
    @toggle-maximise="emit('toggle-maximise')"
    @close="emit('close')"
  />
</template>
