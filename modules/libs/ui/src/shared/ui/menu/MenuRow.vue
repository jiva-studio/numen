<script setup lang="ts">
/**
 * One item of a menu: what stands above it where it begins a group, what is
 * drawn before its words, the words, and what is said about it underneath.
 *
 * A menu naming an item in force offers a choice between its items, and each
 * row says whether it is the one.
 */
import { useTemplateRef } from 'vue'
import type { GroupedItem } from './item'

defineProps<{
  item: GroupedItem
  /** Which item is the one in force, by the caller's identifier. */
  current: string | null
  /** Whether the name of the group it begins is drawn over it. */
  hasGroupName: boolean
  /** Whether the menu draws icons, whose room is kept on every row. */
  hasIcons: boolean
}>()

defineEmits<{
  /** The row was pressed. */
  (event: 'choose'): void
  /** The keyboard came onto the row. */
  (event: 'focus'): void
}>()

defineSlots<{
  /** What is drawn before the words. */
  default(): unknown
}>()

const row = useTemplateRef<HTMLElement>('row')

/** The keyboard onto this row, which the menu asks for. */
const focus = (): void => row.value?.focus()

defineExpose({ focus })
</script>

<template>
  <p v-if="hasGroupName" class="menu__group-name text-hushed px-2 py-1" aria-hidden="true">
    {{ item.group }}
  </p>
  <hr v-else-if="item.isRule" class="menu__rule" role="separator" />

  <button
    ref="row"
    class="menu__item rounded-node flex w-full items-center px-2 py-1.5 text-left"
    type="button"
    :role="current === null ? 'menuitem' : 'menuitemradio'"
    :aria-checked="current === null ? undefined : item.id === current"
    tabindex="-1"
    :disabled="item.disabled"
    @focus="$emit('focus')"
    @click="$emit('choose')"
  >
    <span v-if="hasIcons" class="menu__icon flex shrink-0 items-center">
      <slot />
    </span>
    <span class="flex min-w-0 flex-col">
      <span class="menu__text">{{ item.text }}</span>
      <span v-if="item.detail" class="menu__detail">{{ item.detail }}</span>
    </span>
  </button>
</template>

<style scoped>
/* One physical line, so it stays a hairline however large the interface is
   drawn. */
.menu__rule {
  /* The room a rule keeps on each side of itself. */
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

/* The room an icon takes, kept whether or not the item draws one, so the words
   line up down the menu. What is drawn in it is the caller's. */
.menu__icon {
  /* How large an icon is drawn, and the room between it and the words. */
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

/* What an item is beside its words: the address a model is fetched from, the
   place a file stands. */
.menu__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
