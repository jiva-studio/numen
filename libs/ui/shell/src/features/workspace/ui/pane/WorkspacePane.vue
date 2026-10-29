<script setup lang="ts">
/**
 * A stack of tabs and the one of them that is showing.
 *
 * The strip and the pane carry their ids on the element, so a drag can find
 * what is under the pointer by querying the document.
 */
import { onMounted, useTemplateRef, watch } from 'vue'
import { PaneBody } from './body'
import { PaneStrip } from './strip'
import type { Pane, TabId } from '../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    pane: Pane
    /** The pane a tab would open into. */
    isFocused?: boolean
  }>(),
  { isFocused: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, pointer: PointerEvent): void
  /** A context menu requested on a tab, with its screen position. */
  (event: 'menu', tab: TabId, pos: { x: number; y: number }): void
  /** This pane requests to become the target for newly opened tabs. */
  (event: 'claim'): void
  /** The tab that is now the one showing, once it is on screen. */
  (event: 'show', tab: TabId): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/* ---------------------------------- State --------------------------------- */
const strip = useTemplateRef<InstanceType<typeof PaneStrip>>('strip')

/* ---------------------------------- Hooks --------------------------------- */
/**
 * A hidden tab is rendered with no size, so `show` is emitted once it is
 * visible and can be measured. It is also emitted for the tab this pane opens
 * with.
 */
onMounted(() => {
  if (props.pane.active !== null) emit('show', props.pane.active)
})

watch(
  () => props.pane.active,
  (tab) => {
    if (tab !== null) emit('show', tab)
  },
  { flush: 'post' },
)

/* -------------------------------- Handlers -------------------------------- */
/** A pane is claimed under the primary button and under no other. */
function onClaim(event: PointerEvent): void {
  if (event.button === 0) emit('claim')
}

/** A panel release puts the keyboard back on the tab it is held under. */
function onLeave(tab: TabId): void {
  strip.value?.reach(tab)
}
</script>

<template>
  <section
    class="pane numen bg-surface text-ink flex min-h-0 min-w-0 flex-col"
    :data-workspace-pane="pane.id"
    :data-focused="isFocused || undefined"
    @pointerdown="onClaim"
  >
    <!-- The strip is the list of tabs. A pane holding no tabs has none. -->
    <PaneStrip
      v-if="pane.tabs.length > 0"
      ref="strip"
      :pane="pane"
      :is-focused="isFocused"
      @choose="emit('choose', $event)"
      @close="emit('close', $event)"
      @lift="(tab, at) => emit('lift', tab, at)"
      @menu="(tab, at) => emit('menu', tab, at)"
    >
      <template v-if="$slots.icon" #icon="bound">
        <slot name="icon" v-bind="bound" />
      </template>

      <template v-if="$slots.mark" #mark="bound">
        <slot name="mark" v-bind="bound" />
      </template>
    </PaneStrip>

    <PaneBody :pane="pane" @leave="onLeave">
      <template #tab="bound">
        <slot name="tab" v-bind="bound" />
      </template>

      <template v-if="$slots.silence" #silence>
        <slot name="silence" />
      </template>
    </PaneBody>
  </section>
</template>

<style scoped>
.pane {
  block-size: 100%;
}
</style>
