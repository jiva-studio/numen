<script setup lang="ts">
import { useTemplateRef } from 'vue'
import type { TabId } from '../../lib/node'
/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    tab: TabId
    title: string
    mark?: string | undefined
    isShowing?: boolean
    isFocused?: boolean
  }>(),
  { mark: undefined, isShowing: false, isFocused: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'press', pointer: PointerEvent): void
  (event: 'close'): void
  (event: 'menu', pos: { x: number; y: number }): void
}>()

defineSlots<{
  icon(): unknown
  mark(props: { mark: string }): unknown
}>()

/* ---------------------------------- State --------------------------------- */
const root = useTemplateRef<HTMLElement>('root')

/* -------------------------------- Handlers -------------------------------- */
const onPointerDown = (event: PointerEvent) => {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()
  emit('press', event)
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
    style="--wails-draggable: no-drag"
    @pointerdown="onPointerDown"
    @contextmenu.prevent="onContextMenu"
  >
    <span v-if="$slots.icon" class="tab__icon flex shrink-0 items-center">
      <slot name="icon" />
    </span>

    <span class="min-w-0 truncate" :title="title">{{ title }}</span>

    <slot v-if="mark" name="mark" :mark="mark">
      <span class="tab__mark shrink-0" role="img" :aria-label="mark" :title="mark" />
    </slot>

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
  --height: 1.4rem;
  --pad: 0.4rem;
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

.tab[data-showing] {
  background: var(--numen-raised);
  color: var(--numen-ink);
}

.tab__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
}

.tab__mark {
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: 50%;
  background: currentColor;
}

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
