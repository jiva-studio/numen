<script setup lang="ts">
/**
 * The view a plugin registered for the slot `titleBar`, drawn with the stylesheet of its plugin and
 * given what the window knows of the bar. Without a view it draws nothing.
 */
import { computed, type Component } from 'vue'
import { getPluginStyle, type DeclaredSlotView, type TitleBarViewProps } from '@numen/plugin-host'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<
  Omit<TitleBarViewProps, 'pluginId'> & {
    /** The view of the slot, or null while no plugin draws the bar. */
    view: DeclaredSlotView | null
  }
>()

const emit = defineEmits<{
  (event: 'minimize-window'): void
  (event: 'toggle-maximize-window'): void
  (event: 'close-window'): void
}>()

/* --------------------------------- State ---------------------------------- */
const style = computed(() => (props.view ? getPluginStyle(props.view.pluginId) : undefined))
</script>

<template>
  <template v-if="view">
    <link v-if="style" rel="stylesheet" :href="style" />
    <component
      :is="view.registration.component as Component"
      :key="view.pluginId"
      :plugin-id="view.pluginId"
      :app-name="appName"
      :vault-name="vaultName"
      :platform="platform"
      :is-maximized="isMaximized"
      :is-fullscreen="isFullscreen"
      :commands="commands"
      :vaults="vaults"
      @minimize-window="emit('minimize-window')"
      @toggle-maximize-window="emit('toggle-maximize-window')"
      @close-window="emit('close-window')"
    />
  </template>
</template>
