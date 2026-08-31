<script setup lang="ts">
/**
 * One value along a track, moved by a handle. The track fills with the accent
 * behind the handle.
 *
 * The handle is the stop on the way round the screen: the arrow keys move it a
 * step, and home and end take it to the ends. It draws no number of its own,
 * so what it stands at is read out beside it.
 *
 * It stands inside the ends: a value past one of them, and a value the ends
 * move under, are brought in and handed on.
 *
 * Moving it and letting it go are two things said, so a caller can follow the
 * handle while it moves and act once it has come to rest.
 */
import { computed, watch, type HTMLAttributes } from 'vue'
import { SliderRange, SliderRoot, SliderThumb, SliderTrack } from 'reka-ui'
import { cn } from '@/lib/utils'
import { clamped, type Bounds } from './track'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** How far the track runs, and what one step of it moves. */
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { min: 0, max: 100, step: 1, disabled: false },
)

/** Where the handle stands. */
const model = defineModel<number>({ default: 0 })

const raises = defineEmits<{
  /** The handle let go of, at the end of a drag or of a walk with the keys. */
  settles: [value: number]
}>()

/**
 * The last value handed on, so a value that arrives twice — once as the handle
 * moving and once as it coming to rest — is handed on once.
 */
let handed = model.value

watch(model, (now) => {
  handed = now
})

const hands = (said: number) => {
  if (said === handed) return
  handed = said
  model.value = said
}

const bounds = computed<Bounds>(() => ({ min: props.min, max: props.max, step: props.step }))

/** Where the handle stands, which is inside the ends whatever it was given. */
const standing = computed(() => clamped(model.value, bounds.value))

watch(standing, hands, { immediate: true })

const moved = (value: number[] | undefined) => {
  const said = value?.[0]
  if (typeof said === 'number') hands(said)
}

/**
 * The handle come to rest. What it came to rest at is handed on before it is
 * said to have settled, so a caller acting on the second has the first.
 */
const settled = (value: number[]) => {
  const said = value[0]
  if (typeof said !== 'number') return
  hands(said)
  raises('settles', said)
}
</script>

<template>
  <SliderRoot
    data-slot="slider"
    orientation="horizontal"
    :model-value="[standing]"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    :class="
      cn(
        'relative flex w-full touch-none select-none items-center',
        'data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50',
        props.class,
      )
    "
    @update:model-value="moved"
    @value-commit="settled"
  >
    <SliderTrack class="relative h-1 w-full grow rounded-pill bg-hushed">
      <SliderRange class="absolute h-full rounded-pill bg-accent" />
    </SliderTrack>
    <SliderThumb
      v-bind="$attrs"
      :class="
        cn(
          'block size-4 shrink-0 rounded-pill border border-rule bg-raised',
          'cursor-pointer transition-colors duration-100 ease-numen',
          'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
          'data-[disabled]:cursor-not-allowed',
        )
      "
    />
  </SliderRoot>
</template>
