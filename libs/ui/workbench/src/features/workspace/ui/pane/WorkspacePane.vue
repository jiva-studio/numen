<script setup lang="ts">
import { onMounted, useTemplateRef, watch, type VNode } from 'vue'
import { PaneBody } from './body'
import { PaneStrip } from './strip'
import type { Pane, TabId } from '../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    pane: Pane
    isFocused?: boolean
  }>(),
  { isFocused: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'press', tab: TabId, pointer: PointerEvent): void
  (event: 'menu', tab: TabId, pos: { x: number; y: number }): void
  (event: 'claim'): void
  (event: 'show', tab: TabId): void
}>()

defineSlots<{
  tab(props: { id: TabId }): VNode[]
  icon(props: { id: TabId }): VNode[]
  mark(props: { id: TabId; mark: string }): VNode[]
  silence(): VNode[]
}>()

/* ---------------------------------- State --------------------------------- */
const strip = useTemplateRef<InstanceType<typeof PaneStrip>>('strip')

/* ---------------------------------- Hooks --------------------------------- */
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
function onClaim(event: PointerEvent): void {
  if (event.button === 0) emit('claim')
}

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
    <PaneStrip
      v-if="pane.tabs.length > 0"
      ref="strip"
      :pane="pane"
      :is-focused="isFocused"
      @choose="emit('choose', $event)"
      @close="emit('close', $event)"
      @press="(tab, at) => emit('press', tab, at)"
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
