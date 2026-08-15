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
  }>(),
  { showEdgeLabels: true, duration: 420, environment: () => browserEnvironment },
)

const emit = defineEmits<{
  /** A node other than the focus was chosen, by click or by keyboard. */
  (event: 'activate', id: string): void
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
      :show-edge-labels="showEdgeLabels"
      @activate="emit('activate', $event)"
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
