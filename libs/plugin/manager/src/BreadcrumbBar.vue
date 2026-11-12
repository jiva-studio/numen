<script setup lang="ts">
/**
 * The bar at the top of a settings page: a back button and the trail from Settings to the page.
 * The bar stays at the top while the page scrolls.
 */
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  current: string
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
  <div class="breadcrumb-bar">
    <button
      type="button"
      class="breadcrumb-bar__back"
      :aria-label="words.backToPlugins"
      @click="onBack"
    >
      <ChevronLeft :size="16" aria-hidden="true" />
    </button>
    <nav class="breadcrumb-bar__trail" :aria-label="words.breadcrumb">
      <button type="button" class="breadcrumb-bar__crumb" @click="onBack">
        {{ words.settings }}
      </button>
      <ChevronRight :size="12" aria-hidden="true" />
      <span class="breadcrumb-bar__current" aria-current="page">{{ props.current }}</span>
    </nav>
  </div>
</template>

<style scoped>
.breadcrumb-bar {
  position: sticky;
  inset-block-start: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-block: 0.5rem;
  background-color: var(--numen-surface);
}

.breadcrumb-bar__back {
  display: grid;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.breadcrumb-bar__back:hover,
.breadcrumb-bar__crumb:hover {
  background-color: var(--numen-bubble-bg);
  color: var(--numen-ink);
}

.breadcrumb-bar__back:focus-visible,
.breadcrumb-bar__crumb:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.breadcrumb-bar__trail {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-inline-size: 0;
  color: var(--numen-hushed);
}

.breadcrumb-bar__crumb {
  padding: 0.125rem 0.25rem;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: inherit;
  font-family: inherit;
  font-size: var(--numen-text-2);
  cursor: pointer;
}

.breadcrumb-bar__current {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
