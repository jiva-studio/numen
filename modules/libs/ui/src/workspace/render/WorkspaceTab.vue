<script setup lang="ts">
/**
 * One tab in a strip: what it is called, what it is carrying, whether it is
 * the one showing, and the handle it is dragged by.
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
    /** What the tab is carrying besides its title, in a word. */
    mark?: string | undefined
    /** The one its pane is showing. */
    showing?: boolean
    /** Showing, in the pane a tab would open into. */
    focused?: boolean
  }>(),
  { mark: undefined, showing: false, focused: false },
)

const emit = defineEmits<{
  (event: 'lift', at: PointerEvent): void
  (event: 'close'): void
}>()

defineSlots<{
  /** What a mark is drawn as. Given one, the caller draws its own. */
  mark(props: { mark: string }): unknown
}>()

/**
 * A tab is lifted under the primary button and under no other. The press
 * selects no text as it travels, and takes the keyboard itself.
 */
const onPointerDown = (event: PointerEvent) => {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()
  emit('lift', event)
}
</script>

<template>
  <div
    class="tab numen flex min-w-0 max-w-56 shrink items-center font-sans text-small text-hushed"
    role="tab"
    :aria-selected="showing"
    :tabindex="showing ? 0 : -1"
    :data-workspace-tab="tab"
    :data-showing="showing || undefined"
    :data-focused="focused || undefined"
    @pointerdown="onPointerDown"
  >
    <!-- The whole name is on the element, for a title too long to be drawn. -->
    <span class="min-w-0 truncate" :title="title">{{ title }}</span>

    <slot v-if="mark" name="mark" :mark="mark">
      <span class="tab__mark shrink-0" role="img" :aria-label="mark" :title="mark" />
    </slot>

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
  /* How tall a strip stands, and the room at its edges. The room between the
     name and the cross is the same, so the cross stands with equal air on
     either side of it. */
  --height: 1.4rem;
  --pad: 0.5rem;

  block-size: var(--height);
  padding-inline: var(--pad);
  column-gap: var(--pad);
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}

/* The one showing is raised, as a node in focus is. */
.tab[data-showing] {
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
}

/* What the tab is carrying, drawn as a dot in the colour of the text. */
.tab__mark {
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: 50%;
  background: currentColor;
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
