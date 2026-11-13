<script setup lang="ts">
/** A settings page with its toolbar as the first line; both scroll together. */
import type { VNode } from 'vue'
import SettingsToolbar from './SettingsToolbar.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  path: readonly string[]
}>()

const emit = defineEmits<{
  (e: 'back'): void
  /** An earlier segment was chosen, by its index in the path. */
  (e: 'open', level: number): void
}>()

defineSlots<{
  /** The actions of the page, at the right of the toolbar. */
  actions?(): VNode[]
  /** The page. */
  default(): VNode[]
}>()
</script>

<template>
  <div class="settings-page">
    <SettingsToolbar :path="props.path" @back="emit('back')" @open="(level) => emit('open', level)">
      <template v-if="$slots.actions" #default>
        <slot name="actions" />
      </template>
    </SettingsToolbar>
    <slot />
  </div>
</template>

<style scoped>
.settings-page {
  container: settings / inline-size;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  box-sizing: border-box;
  block-size: 100%;
  min-block-size: 0;
  overflow-y: auto;
  padding: 0 max(var(--numen-gutter), calc((100% - var(--settings-measure)) / 2))
    var(--numen-gutter);
}
</style>
