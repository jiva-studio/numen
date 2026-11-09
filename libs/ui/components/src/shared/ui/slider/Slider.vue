<script setup lang="ts">
/**
 * One value along a track, moved by a handle, with the track filled behind it.
 * It draws no number of its own.
 *
 * A value outside the bounds is clamped and emitted. Moving the handle and
 * releasing it are separate events, and a key sequence ends when the key is
 * released.
 */
import { computed, ref, watch, type HTMLAttributes } from 'vue'
import { SliderRange, SliderRoot, SliderThumb, SliderTrack } from 'reka-ui'
import { cn } from '@/shared/lib/dom/classes'
import { clamp, type Bounds } from '@/shared/lib/math/bounds'
import { isWalkingKey, stepForKey } from './track'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    min?: number
    max?: number
    step?: number
    isDisabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { min: 0, max: 100, step: 1, isDisabled: false },
)

/* --------------------------------- Events --------------------------------- */
/** The handle position. */
const model = defineModel<number>({ default: 0 })

const emit = defineEmits<{
  commit: [value: number]
}>()

/* ---------------------------------- State --------------------------------- */
const bounds = computed<Bounds>(() => ({ min: props.min, max: props.max, step: props.step }))

/**
 * The last value handed on, so a value that arrives twice — once as the handle
 * moving and once as it coming to rest — is handed on once.
 */
let handed = model.value

/** The handle position: the person's while the handle is being moved, else the caller's, clamped. */
const position = ref(clamp(model.value, bounds.value))

/** Whether a key is pressed, and the handle position when it was pressed. */
let walking = false
let began = 0

/** Whether the pointer is down on the handle or the track. */
let dragging = false

const isMoving = () => walking || dragging

const setValue = (value: number) => {
  position.value = value
  if (value === handed) return
  handed = value
  model.value = value
}

watch(model, (now) => {
  if (!isMoving()) handed = now
})

watch(
  () => clamp(model.value, bounds.value),
  (now) => {
    if (!isMoving()) setValue(now)
  },
  { immediate: true },
)

/* -------------------------------- Handlers -------------------------------- */
const onMove = (value: number[] | undefined) => {
  const next = value?.[0]
  if (typeof next === 'number') setValue(next)
}

const onKeyDown = (event: KeyboardEvent) => {
  if (!isWalkingKey(event.key)) return
  event.preventDefault()
  event.stopPropagation()
  if (props.isDisabled) return
  if (!walking) {
    walking = true
    began = handed
  }
  const next = stepForKey(event.key, position.value, {
    bounds: bounds.value,
    isBigStep: event.shiftKey,
  })
  if (next !== null) setValue(next)
}

const onRelease = () => {
  if (!walking) return
  walking = false
  if (handed !== began) emit('commit', handed)
}

const onCommit = (value: number[]) => {
  const next = value[0]
  if (typeof next !== 'number') return
  setValue(next)
  emit('commit', next)
}

const onPointerDown = () => {
  if (!props.isDisabled) dragging = true
}

const onPointerEnd = () => {
  dragging = false
}
</script>

<template>
  <SliderRoot
    data-slot="slider"
    orientation="horizontal"
    :model-value="[position]"
    :min="min"
    :max="max"
    :step="step"
    :disabled="isDisabled"
    :class="
      cn(
        'relative flex w-full touch-none items-center select-none',
        'data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50',
        props.class,
      )
    "
    @pointerdown="onPointerDown"
    @pointerup="onPointerEnd"
    @pointercancel="onPointerEnd"
    @lostpointercapture="onPointerEnd"
    @update:model-value="onMove"
    @value-commit="onCommit"
  >
    <SliderTrack class="rounded-pill bg-hushed relative h-1 w-full grow">
      <SliderRange class="rounded-pill bg-accent absolute h-full" />
    </SliderTrack>
    <SliderThumb
      v-bind="$attrs"
      :class="
        cn(
          'rounded-pill border-rule bg-raised block size-4 shrink-0 border',
          'duration-hover ease-numen cursor-pointer transition-colors',
          'ring-numen outline-none',
          'data-[disabled]:cursor-not-allowed',
        )
      "
      @keydown="onKeyDown"
      @keyup="onRelease"
      @blur="onRelease"
    />
  </SliderRoot>
</template>
