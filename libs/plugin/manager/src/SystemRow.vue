<script setup lang="ts">
/**
 * An entry row of the settings list that opens a page of settings: System by default, others by props.
 */
import { PLUGIN_WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<{ name?: string; detail?: string }>(), {
  name: words.systemSettings,
  detail: words.systemSettingsDetail,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'select'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onClick(): void {
  emit('select')
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    emit('select')
  }
}
</script>

<template>
  <div
    class="plugin-row system-row"
    role="button"
    tabindex="0"
    :aria-label="props.name"
    @click="onClick"
    @keydown="onKeydown"
  >
    <div class="plugin-row__main">
      <div class="plugin-row__header">
        <span class="plugin-row__name">{{ props.name }}</span>
      </div>
      <p v-if="props.detail" class="plugin-row__description">{{ props.detail }}</p>
    </div>

    <div class="plugin-row__trail">
      <span class="plugin-row__chevron" aria-hidden="true">&rsaquo;</span>
    </div>
  </div>
</template>

<style scoped>
.plugin-row {
  --row-min-size: 3.5rem;
  --row-pad-block: 0.625rem;
  --row-pad-inline: 0.75rem;
  --row-text-gap: 0.125rem;
  --row-header-gap: 0.5rem;
  --focus-width: 0.125rem;
  --focus-inset: -0.125rem;
  --chevron-size: 1.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  min-block-size: var(--row-min-size);
  padding: var(--row-pad-block) var(--row-pad-inline);
  cursor: pointer;
  user-select: none;
  transition:
    background-color 0.15s ease,
    opacity 0.15s ease;
  border-radius: var(--numen-radius-tight);
}

.plugin-row:hover {
  background-color: var(--numen-bubble-bg);
}

.plugin-row:focus-visible {
  outline: var(--focus-width) solid var(--numen-accent);
  outline-offset: var(--focus-inset);
}

.plugin-row__main {
  display: flex;
  flex-direction: column;
  gap: var(--row-text-gap);
  flex: 1 1 auto;
  min-inline-size: 0;
}

.plugin-row__header {
  display: flex;
  align-items: baseline;
  gap: var(--row-header-gap);
}

.plugin-row__name {
  color: var(--numen-ink);
  font-weight: 500;
  font-size: var(--numen-text-2);
}

.plugin-row__description {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-inline-size: 100%;
}

.plugin-row__trail {
  display: flex;
  align-items: center;
  gap: var(--numen-panel-gap);
  flex-shrink: 0;
}

.plugin-row__chevron {
  font-size: var(--chevron-size);
  line-height: 1;
  color: var(--numen-hushed);
}
</style>
