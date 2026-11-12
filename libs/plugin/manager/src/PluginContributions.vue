<script setup lang="ts">
/**
 * What a plugin adds to the window, one row per kind: its commands with their shortcuts, its tabs,
 * the files it opens, the actions on files, what it runs by itself and where it appears.
 */
import { KeyCap, type Keystroke } from '@numen/ui'
import type { PluginContribution } from './pluginInfo'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ contributions: readonly PluginContribution[] }>()

/* -------------------------------- Helpers --------------------------------- */
const IS_APPLE = typeof navigator !== 'undefined' && /mac|iphone|ipad/i.test(navigator.userAgent)

/** The keystroke a manifest shortcut such as `mod+shift+a` is drawn as. */
function getKeystroke(keys: string): Keystroke {
  const parts = keys.split('+')
  const icons = parts.slice(0, -1).flatMap((one) => {
    if (one === 'mod') return [IS_APPLE ? ('command' as const) : ('control' as const)]
    if (one === 'shift') return ['shift' as const]
    if (one === 'alt') return ['option' as const]
    if (one === 'ctrl') return ['control' as const]
    return []
  })
  return { icons, letter: (parts[parts.length - 1] ?? '').toUpperCase() }
}
</script>

<template>
  <div class="plugin-contributions">
    <div v-for="group in props.contributions" :key="group.kind" class="plugin-contributions__row">
      <span class="plugin-contributions__kind">{{ group.kind }}</span>
      <span class="plugin-contributions__items">
        <span v-for="item in group.items" :key="item.text" class="plugin-contributions__item">
          <span>{{ item.text }}</span>
          <code v-if="item.detail && item.detail !== item.text" class="plugin-contributions__detail">{{ item.detail }}</code>
          <KeyCap v-if="item.keys" :keys="getKeystroke(item.keys)" />
        </span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.plugin-contributions {
  padding-block: calc(var(--settings-card-pad, 1rem) - var(--settings-row-pad, 0.75rem));
  padding-inline: var(--settings-card-pad, 1rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-contributions__row {
  display: grid;
  grid-template-columns: 10rem minmax(0, 1fr);
  align-items: baseline;
  align-content: center;
  gap: 1rem;
  box-sizing: border-box;
  min-block-size: var(--settings-row-min, 3.25rem);
  padding-block: var(--settings-row-pad, 0.625rem);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.plugin-contributions__row:last-child {
  border-block-end: none;
}

.plugin-contributions__kind {
  color: var(--numen-hushed);
}

.plugin-contributions__items {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  min-inline-size: 0;
}

.plugin-contributions__item {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem 0.625rem;
  color: var(--numen-ink);
}

.plugin-contributions__detail {
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

@container (max-width: 34rem) {
  .plugin-contributions__row {
    grid-template-columns: minmax(0, 1fr);
    gap: 0.25rem;
  }
}
</style>
