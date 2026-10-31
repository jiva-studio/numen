<script setup lang="ts">
/**
 * System settings entry row in the settings list: window appearance, theme, scale.
 */
import { PLUGIN_WORDS as words } from './words'

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
    :aria-label="words.systemSettings"
    @click="onClick"
    @keydown="onKeydown"
  >
    <div class="plugin-row__main">
      <div class="plugin-row__header">
        <span class="plugin-row__name">{{ words.systemSettings }}</span>
      </div>
      <p class="plugin-row__description">
        {{ words.systemSettingsDetail }}
      </p>
    </div>

    <div class="plugin-row__trail">
      <span class="plugin-row__chevron" aria-hidden="true">&rsaquo;</span>
    </div>
  </div>
</template>

<style scoped>
.plugin-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  min-block-size: 3.5rem;
  padding: 0.625rem 0.75rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  cursor: pointer;
  user-select: none;
  transition:
    background-color 0.15s ease,
    opacity 0.15s ease;
  border-radius: var(--numen-radius-small);
}

.plugin-row:last-child {
  border-block-end: none;
}

.plugin-row:hover {
  background-color: var(--numen-surface-hover);
}

.plugin-row:focus-visible {
  outline: 2px solid var(--numen-accent);
  outline-offset: -2px;
}

.plugin-row__main {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  flex: 1 1 auto;
  min-inline-size: 0;
}

.plugin-row__header {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
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
  font-size: 1.25rem;
  line-height: 1;
  color: var(--numen-hushed);
}
</style>
