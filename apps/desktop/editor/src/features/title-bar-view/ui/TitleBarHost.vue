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
  (event: 'run-command', id: string): void
  (event: 'switch-vault', id: string): void
  (event: 'minimize-window'): void
  (event: 'toggle-maximize-window'): void
  (event: 'close-window'): void
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
    :platform="platform"
    :is-maximized="isMaximized"
    :is-fullscreen="isFullscreen"
    :commands="commands"
    :vaults="vaults"
    @run-command="(id: string) => emit('run-command', id)"
    @switch-vault="(id: string) => emit('switch-vault', id)"
    @minimize-window="emit('minimize-window')"
    @toggle-maximize-window="emit('toggle-maximize-window')"
    @close-window="emit('close-window')"
  />
</template>
