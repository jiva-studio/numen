<script setup lang="ts">
/**
 * The plex: one node in focus, everything else placed by its role.
 *
 * Takes a neighbourhood, gives back an identifier when one is chosen. It does
 * not fetch, does not know what an identifier addresses, and does not work out
 * who is related to whom. Answer `activate` with the next neighbourhood and it
 * travels there by itself.
 */
import { computed, onMounted, onScopeDispose, ref, toRef, useTemplateRef } from 'vue'
import PlexView from './render/PlexView.vue'
import { usePlexTransition, browserEnvironment, type Environment } from './transition'
import type { Placement, PlexOptionsInput } from './arrange'
import { countOf, type PlexNeighbourhood, type PlexRelatedRole } from './model'
import { resolveOptions } from './arrange'
import { usePlexGesture } from './gesture'

const props = withDefaults(
  defineProps<{
    neighbourhood: PlexNeighbourhood
    options?: PlexOptionsInput
    /** Rows and columns unless another arrangement is handed in. */
    placement?: Placement
    showEdgeLabels?: boolean
    /** Milliseconds. Zero arrives instantly, as does reduced motion. */
    duration?: number
    /** The clock. Browser by default; a test hands in its own. */
    environment?: Environment
    /**
     * Seats a gesture may produce. A sibling is another of the parent's
     * children rather than something anyone makes directly, so it is left out
     * — but which relationships exist is the caller's to say, not the plex's.
     */
    creatable?: readonly PlexRelatedRole[]
    /** How far a gesture travels before it is a drag and not a click. */
    dragThreshold?: number
  }>(),
  {
    showEdgeLabels: true,
    duration: 420,
    environment: () => browserEnvironment,
    creatable: () => ['parent', 'child', 'jump'],
    dragThreshold: 8,
  },
)

const emit = defineEmits<{
  /** A node other than the focus was chosen, by click or by keyboard. */
  (event: 'activate', id: string): void
  /** Reached out into empty space: make a node in this seat of that one. */
  (event: 'create', from: string, role: PlexRelatedRole): void
  /** Reached out onto another node: relate the two in this seat. */
  (event: 'link', from: string, to: string, role: PlexRelatedRole): void
}>()

/** Measured here and used twice: to wrap the arrangement, and to centre it. */
const FALLBACK = { width: 1200, height: 800 }

const frameElement = useTemplateRef<HTMLElement>('frame')
const viewport = ref(FALLBACK)

onMounted(() => {
  const element = frameElement.value
  if (!element || typeof ResizeObserver === 'undefined') return

  const observer = new ResizeObserver(([entry]) => {
    const box = entry?.contentRect
    if (box && box.width > 0 && box.height > 0) {
      viewport.value = { width: box.width, height: box.height }
    }
  })
  observer.observe(element)
  onScopeDispose(() => observer.disconnect())
})

const { frame, moving } = usePlexTransition(
  () => props.neighbourhood,
  () => ({
    options: { ...props.options, viewport: viewport.value },
    placement: props.placement,
  }),
  () => props.duration,
  props.environment,
)

const hovered = ref<string | null>(null)

/**
 * Where a handle is worth offering. A caller that allows no seat at all has
 * turned the gesture off, and a handle that can come to nothing is a lie.
 */
const reachable = computed(() => (props.creatable.length ? hovered.value : null))

/** Settled once and read by both the gesture and the drawing. */
const options = computed(() => resolveOptions({ ...props.options, viewport: viewport.value }))

/**
 * Reaching out from a node.
 *
 * The plex works out the shape of it — where it started, which way it went,
 * what it landed on — and says so. Whether a parent may be made, and what
 * making one writes, is the application's.
 */
const gesture = usePlexGesture(
  () => frame.value,
  () => options.value,
  () => props.creatable,
  () => props.dragThreshold,
  (drop) => {
    if (drop.kind === 'create') emit('create', drop.from, drop.role)
    else emit('link', drop.from, drop.to, drop.role)
  },
)

/**
 * What did not fit, as `[role, count]` pairs rather than a sentence — the
 * words belong to whoever renders the plex, through the `overflow` slot.
 */
const overflow = computed(
  () =>
    (Object.entries(frame.value.overflow) as [PlexRelatedRole, number][]).filter(
      ([, count]) => count > 0,
    ),
)

defineExpose({ moving: toRef(moving) })
</script>

<template>
  <div
    ref="frame"
    class="plex-frame numen"
    :data-moving="moving || undefined"
    :style="{ '--numen-plex-move': `${duration}ms` }"
  >
    <PlexView
      :frame="frame"
      :viewport="viewport"
      :node-size="options.nodeSize"
      :show-edge-labels="showEdgeLabels"
      :hovered="reachable"
      :gesture-from="gesture.from.value"
      :gesture-at="gesture.at.value"
      :gesture-outcome="gesture.outcome.value"
      @activate="emit('activate', $event)"
      @hover="hovered = $event"
      @reach="gesture.begin"
    />
    <div v-if="overflow.length" class="plex-frame__overflow" role="status">
      <slot name="overflow" :overflow="overflow">
        {{ overflow.map(([role, count]) => countOf(role, count)).join(', ') }} not shown
      </slot>
    </div>
  </div>
</template>

<style scoped>
.plex-frame {
  position: relative;
  inline-size: 100%;
  block-size: 100%;
  background: var(--numen-surface);
}

.plex-frame__overflow {
  position: absolute;
  inset-block-end: 8px;
  inset-inline-start: 12px;
  color: var(--numen-edge-label);
  font-family: var(--numen-font-sans);
  font-size: 11px;
}
</style>
