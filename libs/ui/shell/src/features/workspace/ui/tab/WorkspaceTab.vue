<script setup lang="ts">
/**
 * One tab in a strip: what it is called, what it is carrying, whether it is
 * the one showing, and the handle it is dragged by.
 *
 * A press reports itself and waits. Whether it turns out to be a choice or the
 * start of a drag is settled by what the pointer does next, which the
 * workspace watches.
 */
import { useTemplateRef } from 'vue'
import type { TabId } from '../../lib/node'
/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    /** Its identity, carried on the element for a drag to find it by. */
    tab: TabId
    title: string
    /** What the tab is carrying besides its title, in a word. */
    mark?: string | undefined
    /** The one its pane is showing. */
    isShowing?: boolean
    /** Showing, in the pane a tab would open into. */
    isFocused?: boolean
  }>(),
  { mark: undefined, isShowing: false, isFocused: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'lift', pointer: PointerEvent): void
  (event: 'close'): void
  /** A menu request on the tab, with the screen position. */
  (event: 'menu', pos: { x: number; y: number }): void
}>()

defineSlots<{
  /** Rendered before the name; shows the type of tab. */
  icon(): unknown
  /** What a mark is drawn as. Given one, the caller draws its own. */
  mark(props: { mark: string }): unknown
}>()

/* ---------------------------------- State --------------------------------- */
const root = useTemplateRef<HTMLElement>('root')

/* -------------------------------- Handlers -------------------------------- */
/**
 * A tab is lifted with the primary button only. The press does not select text
 * while the tab moves, and the tab takes keyboard focus.
 */
const onPointerDown = (event: PointerEvent) => {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()
  emit('lift', event)
}

function onContextMenu(event: MouseEvent): void {
  emit('menu', { x: event.clientX, y: event.clientY })
}

/* --------------------------------- Helpers -------------------------------- */
defineExpose({ focus: () => root.value?.focus() })
</script>

<template>
  <div
    ref="root"
    class="tab numen text-small text-hushed flex min-w-0 shrink items-center font-sans"
    role="tab"
    :aria-selected="isShowing"
    :tabindex="isShowing ? 0 : -1"
    :data-workspace-tab="tab"
    :data-showing="isShowing || undefined"
    :data-focused="isFocused || undefined"
    @pointerdown="onPointerDown"
    @contextmenu.prevent="onContextMenu"
  >
    <span v-if="$slots.icon" class="tab__icon flex shrink-0 items-center">
      <slot name="icon" />
    </span>

    <!-- The whole name is on the element, for a title too long to be drawn. -->
    <span class="min-w-0 truncate" :title="title">{{ title }}</span>

    <slot v-if="mark" name="mark" :mark="mark">
      <span class="tab__mark shrink-0" role="img" :aria-label="mark" :title="mark" />
    </slot>

    <!-- Reached from the tab it stands on, and only from that one: a strip
         walked with the arrows is one stop, not two for every tab in it. -->
    <button
      class="tab__close rounded-pill shrink-0"
      type="button"
      :tabindex="isShowing ? 0 : -1"
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
  /* Strip height and edge padding. The space between the name and the close
     button is the same, so the close button has equal space on both sides. */
  --height: 1.4rem;
  --pad: 0.4rem;
  /* The maximum width of a tab before its name is truncated. */
  --widest: 14rem;

  block-size: var(--height);
  max-inline-size: var(--widest);
  padding-inline: var(--pad);
  column-gap: var(--pad);
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}

/* The one showing is raised, as a node in focus is. */
.tab[data-showing] {
  background: var(--numen-raised);
  color: var(--numen-ink);
}

/* The kind-of-tab icon, rendered before the name. The row's own gap is the
   space after it, so it has equal space on both sides. */
.tab__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
}

/* What the tab is carrying, drawn as a dot in the colour of the text. */
.tab__mark {
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: 50%;
  background: currentColor;
}

/* The keyboard is drawn at its current position, inside the strip, which is clipped. */
.tab:focus-visible,
.tab__close:focus-visible {
  outline: none;
}

.tab__close {
  inline-size: 1.1em;
  block-size: 1.1em;
  line-height: 1;
  opacity: 0;
}

.tab:hover .tab__close,
.tab[data-showing] .tab__close,
.tab__close:focus-visible {
  opacity: 0.55;
}

.tab__close:hover,
.tab__close:focus-visible {
  opacity: 1;
}
</style>
