<script setup lang="ts">
/**
 * One row of the welcome screen: an icon, a name, a secondary line and a
 * keyboard shortcut.
 *
 * The caller handles activation.
 */
import type { Component } from 'vue'
import { KeyCap } from '@numen/ui'
import type { PaletteKeys } from '@numen/ui'

defineProps<{
  /** The icon before the name. A row without one has no icon. */
  icon?: Component | undefined
  /** The name of the row. */
  text: string
  /** The secondary line under the name. */
  aside?: string | undefined
  /** The full text of the secondary line, for when it is truncated. */
  whole?: string | undefined
  /** The keyboard shortcut of the row, if it has one. */
  keys?: PaletteKeys | undefined
  /** A pending row, which does not respond to interaction. */
  isDisabled?: boolean | undefined
}>()

defineSlots<{
  /** Content shown at the far end of the row. */
  default?(): unknown
}>()
</script>

<template>
  <button type="button" class="welcome-page__row" :disabled="isDisabled">
    <component :is="icon" v-if="icon" class="welcome-page__icon" />
    <span v-if="aside" class="welcome-page__named">
      <span class="welcome-page__what">{{ text }}</span>
      <span class="welcome-page__aside" :title="whole">{{ aside }}</span>
    </span>
    <span v-else class="welcome-page__what">{{ text }}</span>
    <slot />
    <KeyCap v-if="keys" class="welcome-page__keys" :keys="keys" />
  </button>
</template>

<style scoped>
.welcome-page__row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: 100%;
  padding: 0.3rem 0.6rem;
  border: 0;
  border-radius: var(--numen-radius);
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
  cursor: pointer;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.welcome-page__icon {
  flex: none;
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.75;
  opacity: 0.75;
}

/* The name above the secondary line, next to the icon. */
.welcome-page__named {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 0.05rem;
  min-inline-size: 0;
}

.welcome-page__row:hover:not(:disabled) {
  background: var(--numen-bubble-bg);
}

/* A pending row is inert and does not
   answer to a hand. */
.welcome-page__row:disabled {
  cursor: default;
}

.welcome-page__row:focus-visible {
  outline: none;
  outline-offset: 1px;
}

/* A long name ends in an ellipsis.
   It takes the room a row leaves it; inside a name over its second line the
   column above holds the room, and the name takes the height of its own line. */
.welcome-page__what {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.welcome-page__row > .welcome-page__what {
  flex: 1;
}

/* One line, truncated with an ellipsis, since a path can be arbitrarily long. */
.welcome-page__aside {
  overflow: hidden;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Narrow, the row keeps its name and gives up the keystroke drawn on it: the
   row is pressed by hand, and the key still works. Narrower, it gives up what
   is said under the name; a path is on the row itself, for pointing at. */
@container (max-width: 24rem) {
  .welcome-page__keys {
    display: none;
  }
}

@container (max-width: 18rem) {
  .welcome-page__aside {
    display: none;
  }
}
</style>
