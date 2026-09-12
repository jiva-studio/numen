<script setup lang="ts">
/**
 * One value along a track, moved by a handle, with the track filled behind it.
 * It draws no number of its own.
 *
 * It stands inside the ends: a value past one of them, or one the ends move
 * under, is brought in and handed on. Moving the handle and letting it go are
 * two things said, and a walk with the keys is over when the key is.
 */
import { computed, watch, type HTMLAttributes } from 'vue'
import { SliderRange, SliderRoot, SliderThumb, SliderTrack } from 'reka-ui'
import { cn } from '@/shared/lib/classes'
import { clamp, isWalkingKey, stepForKey, type Bounds } from './track'

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

const setValue = (said: number) => {
  if (said === handed) return
  handed = said
  model.value = said
}

const bounds = computed<Bounds>(() => ({ min: props.min, max: props.max, step: props.step }))

/** Where the handle stands, which is inside the ends whatever it was given. */
const inForce = computed(() => clamp(model.value, bounds.value))

watch(inForce, setValue, { immediate: true })

const onMove = (value: number[] | undefined) => {
  const said = value?.[0]
  if (typeof said === 'number') setValue(said)
}

/** Whether a key is down, and where the handle stood when it went down. */
let walking = false
let began = 0

/**
 * The handle taken hold of by the keys, and moved to where the key leaves it. A
 * key held down and a key struck again are one walk, which is over when the key
 * is let go of.
 */
const onKeyDown = (event: KeyboardEvent) => {
  if (!isWalkingKey(event.key)) return
  event.preventDefault()
  event.stopPropagation()
  if (props.disabled) return
  if (!walking) {
    walking = true
    began = handed
  }
  const said = stepForKey(event.key, inForce.value, bounds.value, event.shiftKey)
  if (said !== null) setValue(said)
}

/** The handle let go of, at what the walk left it standing at. */
const onRelease = () => {
  if (!walking) return
  walking = false
  if (handed !== began) raises('settles', handed)
}

/**
 * The handle come to rest under the pointer. What it came to rest at is handed
 * on before it is said to have settled, so a caller acting on the second has
 * the first.
 */
const onCommit = (value: number[]) => {
  const said = value[0]
  if (typeof said !== 'number') return
  setValue(said)
  raises('settles', said)
}
</script>

<template>
  <SliderRoot
    data-slot="slider"
    orientation="horizontal"
    :model-value="[inForce]"
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
    @update:model-value="onMove"
    @value-commit="onCommit"
  >
    <SliderTrack class="relative h-1 w-full grow rounded-pill bg-hushed">
      <SliderRange class="absolute h-full rounded-pill bg-accent" />
    </SliderTrack>
    <SliderThumb
      v-bind="$attrs"
      :class="
        cn(
          'block size-4 shrink-0 rounded-pill border border-rule bg-raised',
          'cursor-pointer transition-colors duration-hover ease-numen',
          'outline-none ring-numen',
          'data-[disabled]:cursor-not-allowed',
        )
      "
      @keydown="onKeyDown"
      @keyup="onRelease"
      @blur="onRelease"
    />
  </SliderRoot>
</template>
