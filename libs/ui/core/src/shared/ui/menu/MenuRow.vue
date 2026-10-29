<script setup lang="ts">
/**
 * One item of a menu: the group rule and name when it begins a group, the leading
 * icon, the label, and the description below it.
 *
 * A menu with a current item offers a choice between its items, and each row
 * indicates whether it is the current one.
 */
import { useTemplateRef } from 'vue'
import type { GroupedItem } from './items/item'

defineProps<{
  item: GroupedItem
  /** Which item is the one in force, by the caller's identifier. */
  current: string | null
  /** Whether the name of the group it begins is drawn over it. */
  hasName: boolean
  /** Whether the menu draws icons, so every row reserves the icon space. */
  hasIcons: boolean
}>()

defineEmits<{
  /** The row was pressed. */
  (event: 'choose'): void
  /** The keyboard came onto the row. */
  (event: 'focus'): void
  /** The pointer moved onto the row. */
  (event: 'hover'): void
}>()

defineSlots<{
  /** What is drawn before the words. */
  default(): unknown
}>()

const row = useTemplateRef<HTMLElement>('row')

/** Focuses this row; called by the menu. */
const focus = (): void => row.value?.focus()

defineExpose({ focus, getElement: () => row.value })
</script>

<template>
  <p v-if="hasName" class="menu__group-name text-hushed px-2 py-1" aria-hidden="true">
    {{ item.group }}
  </p>
  <hr v-else-if="item.isRule" class="menu__rule" role="separator" />

  <button
    ref="row"
    class="menu__item rounded-node flex w-full items-center px-2 py-1.5 text-left"
    type="button"
    :role="current === null ? 'menuitem' : 'menuitemradio'"
    :aria-checked="current === null ? undefined : item.id === current"
    :aria-haspopup="item.children && item.children.length > 0 ? 'menu' : undefined"
    tabindex="-1"
    :disabled="item.isDisabled"
    @focus="$emit('focus')"
    @mouseenter="$emit('hover')"
    @click="$emit('choose')"
  >
    <span v-if="hasIcons" class="menu__icon flex shrink-0 items-center">
      <slot />
    </span>
    <span class="flex min-w-0 flex-1 flex-col">
      <span class="menu__text">{{ item.text }}</span>
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
  box-sizing: border-box;
  max-inline-size: 100%;
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.menu__item:hover:not(:disabled),
.menu__item:focus-visible {
  outline: none;
  background: var(--numen-bubble-bg);
}

.menu__item:disabled {
  color: var(--numen-edge-label);
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
