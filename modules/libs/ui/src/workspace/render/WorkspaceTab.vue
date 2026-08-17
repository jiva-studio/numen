<script setup lang="ts">
/**
 * One tab in a strip: what it is called, whether it is the one showing, and
 * the handle it is dragged by.
 *
 * A press reports itself and waits. Whether it turns out to be a choice or the
 * start of a drag is settled by what the pointer does next, which the
 * workspace watches.
 */
defineProps<{
  title: string
  /** The one its group is showing. */
  showing?: boolean
}>()

const emit = defineEmits<{
  (event: 'lift', at: PointerEvent): void
  (event: 'close'): void
}>()
</script>

<template>
  <div
    class="tab numen flex min-w-0 max-w-56 shrink items-center gap-1.5 px-3 font-sans text-small text-hushed"
    role="tab"
    :aria-selected="showing === true"
    :data-showing="showing || undefined"
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
  /* How tall a strip stands, and how far the tab showing is lifted out of the
     ones beside it. */
  --height: 2.1rem;
  --lift: 1px;

  block-size: var(--height);
  border-inline-end: var(--numen-stroke) solid var(--numen-node-border);
  cursor: default;
  user-select: none;
  touch-action: none;
}

.tab[data-showing] {
  background: var(--numen-surface);
  color: var(--numen-node-fg);
  box-shadow: inset 0 var(--lift) 0 0 var(--numen-focus-bg);
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
