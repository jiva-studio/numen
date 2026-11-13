<script setup lang="ts">
/**
 * A row of a settings list that opens a page: the icon of its category, the name, a one-line
 * detail, the `trail` slot and a chevron (see NavigationRowProps).
 */
import { computed, type VNode } from 'vue'
import { ChevronRight } from '@lucide/vue'
import { getCategoryIcon } from './categoryIcons'
import type { NavigationRowProps } from './NavigationRow.types'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<NavigationRowProps>()

const emit = defineEmits<{
  (e: 'select'): void
}>()

defineSlots<{
  /** Tags drawn before the chevron. */
  trail?(): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const icon = computed(() => getCategoryIcon(props.category))

/* -------------------------------- Handlers -------------------------------- */
function onClick(): void {
  if (!props.isStatic) emit('select')
}
</script>

<template>
  <component
    :is="props.isStatic ? 'div' : 'button'"
    :type="props.isStatic ? undefined : 'button'"
    class="navigation-row"
    :class="{ 'navigation-row--dimmed': props.isDimmed, 'navigation-row--static': props.isStatic }"
    :aria-label="props.label ?? props.name"
    @click="onClick"
  >
    <component
      :is="icon"
      class="navigation-row__glyph"
      :size="20"
      :stroke-width="1.75"
      aria-hidden="true"
    />
    <span class="navigation-row__main">
      <span class="navigation-row__name">{{ props.name }}</span>
      <span v-if="props.detail" class="navigation-row__detail">{{ props.detail }}</span>
    </span>
    <slot name="trail" />
    <ChevronRight
      v-if="!props.isTile && !props.isStatic"
      class="navigation-row__chevron"
      :size="16"
      aria-hidden="true"
    />
  </component>
</template>

<style scoped>
.navigation-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-sizing: border-box;
  inline-size: 100%;
  min-block-size: var(--settings-row-min);
  padding: var(--settings-row-pad) var(--settings-card-pad);
  border: none;
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
  background: none;
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  text-align: start;
  cursor: pointer;
  transition: background-color var(--numen-motion-hover) ease;
}

.navigation-row:first-child {
  border-block-start: none;
}

.navigation-row:hover {
  background-color: var(--settings-hover-bg);
}

.navigation-row--static {
  cursor: default;
}

.navigation-row--static:hover {
  background-color: transparent;
}

.navigation-row:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.navigation-row__glyph,
.navigation-row__chevron {
  flex: none;
  color: var(--numen-hushed);
}

.navigation-row--dimmed .navigation-row__glyph,
.navigation-row--dimmed .navigation-row__name {
  opacity: 0.55;
}

.navigation-row__main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-inline-size: 0;
}

.navigation-row__name {
  font-weight: 500;
}

.navigation-row__detail {
  overflow: hidden;
  color: var(--numen-hushed);
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
