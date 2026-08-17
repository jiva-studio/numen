<script setup lang="ts">
/**
 * One tab in a strip: what it is called, whether it is the one showing, and
 * the handle it is dragged by.
 *
 * A press reports itself and waits. Whether it turns out to be a choice or the
 * start of a drag is settled by what the pointer does next, which the
 * workspace watches.
 */
import type { TabId } from '../model'
withDefaults(
  defineProps<{
    /** Its identity, carried on the element for a drag to find it by. */
    tab: TabId
    title: string
    /** The one its group is showing. */
    showing?: boolean
    /** Showing, in the group a tab would open into. */
    marked?: boolean
  }>(),
  { showing: false, marked: false },
)

const emit = defineEmits<{
  (event: 'lift', at: PointerEvent): void
  (event: 'close'): void
}>()
</script>

<template>
  <div
    class="tab numen flex min-w-0 max-w-56 shrink items-center gap-1.5 px-3 font-sans text-small text-hushed"
    role="tab"
    :aria-selected="showing"
    :data-workspace-tab="tab"
    :data-showing="showing || undefined"
    :data-marked="marked || undefined"
    @pointerdown="emit('lift', $event)"
  >
    <span class="min-w-0 truncate">{{ title }}</span>
    <button
      class="tab__close shrink-0 rounded-pill"
      type="button"
      :aria-label="`Close ${title}`"
      @pointerdown.stop
      @click.stop="emit('close')"
    >
      ×
    </button>
  </div>
</template>

<style scoped>
.tab {
  /* How tall a strip stands, and how thick the mark along the top of the one
     showing is. */
  --height: 2.1rem;
  --lift: 2px;

  block-size: var(--height);
  border-inline-end: var(--numen-stroke) solid var(--numen-node-border);
  cursor: default;
  user-select: none;
  touch-action: none;
}

/* The one showing is raised, as a node in focus is. */
.tab[data-showing] {
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
}

/* The mark along the top belongs to the group a tab would open into. */
.tab[data-marked] {
  box-shadow: inset 0 var(--lift) 0 0 var(--numen-ring);
}

.tab__close {
  inline-size: 1.1em;
  block-size: 1.1em;
  line-height: 1;
  opacity: 0;
}

.tab:hover .tab__close,
.tab[data-showing] .tab__close {
  opacity: 0.55;
}

.tab__close:hover {
  opacity: 1;
}
</style>
