<script setup lang="ts">
/** A settings page under its toolbar; only the page scrolls. */
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
    <div class="settings-page__head">
      <SettingsToolbar
        class="settings-page__measure"
        :path="props.path"
        @back="emit('back')"
        @open="(level) => emit('open', level)"
      >
        <template v-if="$slots.actions" #default>
          <slot name="actions" />
        </template>
      </SettingsToolbar>
    </div>
    <div class="settings-page__body">
      <div class="settings-page__measure settings-page__content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.settings-page__head {
  flex: none;
  padding-inline: var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  background-color: var(--numen-raised);
}

.settings-page__body {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

.settings-page__measure {
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.settings-page__content {
  container: settings / inline-size;
}
</style>
