<script setup lang="ts">
/**
 * A stack of tabs and the one of them that is showing.
 *
 * The strip and the pane carry their identities on the element, so that a
 * drag can find out what the pointer is over by asking the document.
 */
import { onMounted, useTemplateRef, watch } from 'vue'
import { PaneBody } from './body'
import { PaneStrip } from './strip'
import type { Pane, TabId } from '../../lib/node'

const props = withDefaults(
  defineProps<{
    pane: Pane
    /** The pane a tab would open into. */
    isFocused?: boolean
  }>(),
  { isFocused: false },
)

const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, at: PointerEvent): void
  /** This pane asks to be the one a tab opens into. */
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

const strip = useTemplateRef<InstanceType<typeof PaneStrip>>('strip')

/** A pane is claimed under the primary button and under no other. */
function claim(event: PointerEvent): void {
  if (event.button === 0) emit('claim')
}

/** A panel let go of puts the keyboard back on the tab it is held under. */
function onLeave(tab: TabId): void {
  strip.value?.reach(tab)
}

/**
 * A tab held out of sight is drawn with no size, so what is showing is said
 * once it is on screen and can be measured. It is said for the tab this pane
 * opens with as well.
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
</script>

<template>
  <section
    class="pane numen bg-surface text-ink flex min-h-0 min-w-0 flex-col"
    :data-workspace-pane="pane.id"
    :data-focused="isFocused || undefined"
    @pointerdown="claim"
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
