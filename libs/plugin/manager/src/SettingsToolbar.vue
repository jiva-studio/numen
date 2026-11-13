<script setup lang="ts">
/**
 * The bar at the top of a settings page: a back button and the path from Settings to the page on the
 * left, the actions of the page on the right. Each earlier segment of the path opens its level. The
 * bar stays at the top while the page scrolls. The first level, Settings itself, has no back button.
 */
import type { VNode } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** The names of the levels from Settings to the page, the page last. */
  path: readonly string[]
}>()

const emit = defineEmits<{
  (e: 'back'): void
  /** An earlier segment was chosen, by its index in the path. */
  (e: 'open', level: number): void
}>()

defineSlots<{
  /** The actions of the page. */
  default?(): VNode[]
}>()

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onOpen(level: number): void {
  emit('open', level)
}
</script>

<template>
  <div class="settings-toolbar">
    <button
      v-if="props.path.length > 1"
      type="button"
      class="settings-toolbar__back"
      :aria-label="words.back"
      @click="onBack"
    >
      <ChevronLeft :size="16" aria-hidden="true" />
    </button>
    <nav class="settings-toolbar__path" :aria-label="words.path">
      <template v-for="(segment, level) in props.path" :key="level">
        <ChevronRight
          v-if="level > 0"
          class="settings-toolbar__separator"
          :size="12"
          aria-hidden="true"
        />
        <button
          v-if="level < props.path.length - 1"
          type="button"
          class="settings-toolbar__segment"
          @click="onOpen(level)"
        >
          {{ segment }}
        </button>
        <span v-else class="settings-toolbar__current" aria-current="page">{{ segment }}</span>
      </template>
    </nav>
    <div v-if="$slots.default" class="settings-toolbar__actions">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.settings-toolbar {
  position: sticky;
  inset-block-start: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  box-sizing: border-box;
  min-block-size: 3.25rem;
  padding-block: 0.5rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  background-color: var(--numen-surface);
}

.settings-toolbar__back {
  display: grid;
  flex: none;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.settings-toolbar__back:hover,
.settings-toolbar__segment:hover {
  background-color: var(--numen-bubble-bg);
  color: var(--numen-ink);
}

.settings-toolbar__back:focus-visible,
.settings-toolbar__segment:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.settings-toolbar__path {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 0.25rem;
  min-inline-size: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.settings-toolbar__separator {
  flex: none;
}

.settings-toolbar__segment {
  flex: none;
  padding: 0.125rem 0.25rem;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: inherit;
  font-family: inherit;
  font-size: inherit;
  cursor: pointer;
}

.settings-toolbar__current {
  overflow: hidden;
  padding-inline: 0.25rem;
  color: var(--numen-ink);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-toolbar__actions {
  display: flex;
  flex: none;
  align-items: center;
  gap: 0.75rem;
}
</style>
