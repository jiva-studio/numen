<script setup lang="ts">
/**
 * One item of a menu: the group rule and name when it begins a group, the leading
 * icon, the label, and the description below it.
 *
 * A menu with a current item offers a choice between its items, and each row
 * indicates whether it is the current one.
 */
import { useTemplateRef, type VNode } from 'vue'
import type { GroupedItem } from './items/item'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  item: GroupedItem
  current: string | null
  hasName: boolean
  hasIcons: boolean
  isActive: boolean
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  /** The row was pressed. */
  (event: 'choose'): void
  /** The keyboard came onto the row. */
  (event: 'focus'): void
  /** The pointer moved onto or across the row. */
  (event: 'hover', pointer: MouseEvent): void
}>()

defineSlots<{
  /** What is drawn before the words. */
  default(): VNode[]
}>()

defineExpose({ focus, getElement: () => row.value })

/* --------------------------------- State ---------------------------------- */
const row = useTemplateRef<HTMLElement>('row')

/* -------------------------------- Helpers --------------------------------- */
function focus(options?: FocusOptions): void {
  row.value?.focus(options)
}
</script>

<template>
  <p v-if="hasName" class="menu__group-name text-hushed px-2 py-1" aria-hidden="true">
    {{ item.group }}
  </p>
  <hr v-else-if="item.isRule" class="menu__rule" role="separator" />

  <button
    ref="row"
    class="menu__item flex w-full items-center px-2 py-1.5 text-left"
    type="button"
    :role="current === null ? 'menuitem' : 'menuitemradio'"
    :aria-checked="current === null ? undefined : item.id === current"
    :aria-haspopup="item.children && item.children.length > 0 ? 'menu' : undefined"
    :aria-label="item.component ? item.text : undefined"
    tabindex="-1"
    :data-active="isActive ? '' : undefined"
    :disabled="item.isDisabled"
    @focus="$emit('focus')"
    @mouseenter="$emit('hover', $event)"
    @mousemove="$emit('hover', $event)"
    @click="$emit('choose')"
  >
    <span v-if="hasIcons" class="menu__icon flex shrink-0 items-center">
      <slot />
    </span>
    <span class="flex min-w-0 flex-1 flex-col">
      <span class="menu__text">
        <component :is="item.component" v-if="item.component" />
        <template v-else>{{ item.text }}</template>
      </span>
      <span v-if="item.detail" class="menu__detail">{{ item.detail }}</span>
    </span>
    <span
      v-if="item.children && item.children.length > 0"
      class="menu__chevron text-hushed ml-auto pl-2 text-xs"
      aria-hidden="true"
    >
      ›
    </span>
  </button>
</template>

<style scoped>
/* One physical line, so it stays a hairline however large the interface is
   drawn. */
.menu__rule {
  /* The space a rule keeps on each side. */
  --parting: 0.25rem;

  block-size: 0;
  margin-block: var(--parting);
  border: 0;
  border-block-start: 1px solid var(--numen-panel-border);
}

/* The name of a group, set as this product sets a label over what it names. */
.menu__group-name {
  margin: 0;
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.menu__item {
  border-radius: var(--numen-radius-popup-item);
  box-sizing: border-box;
  max-inline-size: 100%;
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.menu__item[data-active]:not(:disabled) {
  outline: none;
  background: var(--numen-bubble-bg);
}

.menu__item:disabled {
  color: var(--numen-edge-label);
}

/* Highlighted and focused rows use the primary text color for detail text. */
.menu__item[data-active]:not(:disabled) .menu__detail {
  color: var(--numen-ink);
}

/* The icon box keeps its size whether or not the item draws an icon, so the words
   line up down the menu. What is drawn in it is the caller's. */
.menu__icon {
  /* Icon size and the gap between the icon and the label. */
  --icon: 0.875rem;
  --icon-gap: 0.5rem;

  inline-size: var(--icon);
  block-size: var(--icon);
  margin-inline-end: var(--icon-gap);
}

/* One line, then an ellipsis. A menu is read down its leading edge. */
.menu__text,
.menu__detail {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
