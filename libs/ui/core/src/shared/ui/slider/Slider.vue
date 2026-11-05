<script setup lang="ts">
/**
 * One value along a track, moved by a handle, with the track filled behind it.
 * It draws no number of its own.
 *
 * A value outside the bounds is clamped and emitted. Moving the handle and
 * releasing it are separate events, and a key sequence ends when the key is
 * released.
 */
import { computed, onBeforeUnmount, ref, watch, type HTMLAttributes } from 'vue'
import { SliderRange, SliderRoot, SliderThumb, SliderTrack } from 'reka-ui'
import { cn } from '@/shared/lib/dom/classes'
import { clamp, type Bounds } from '@/shared/lib/math/bounds'
import { isWalkingKey, stepForKey } from './track'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** The track range and the step size. */
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
  /** The handle was released, at the end of a drag or of a key sequence. */
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
const shown = ref(clamp(model.value, bounds.value))

/** Whether a key is pressed, and the handle position when it was pressed. */
let walking = false
let began = 0

/** Whether the pointer is down on the handle or the track. */
let dragging = false

/** Whether the person is moving the handle: a value from outside does not move it then. */
const isMoving = () => walking || dragging

const setValue = (value: number) => {
  shown.value = value
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

/**
 * The handle taken hold of by the keys, and moved to where the key leaves it. A
 * key held down and a key struck again are one walk, which is over when the key
 * is release.
 */
const onKeyDown = (event: KeyboardEvent) => {
  if (!isWalkingKey(event.key)) return
  event.preventDefault()
  event.stopPropagation()
  if (props.isDisabled) return
  if (!walking) {
    walking = true
    began = handed
  }
  const next = stepForKey(event.key, shown.value, {
    bounds: bounds.value,
    isBigStep: event.shiftKey,
  })
  if (next !== null) setValue(next)
}

/** The handle release, at what the walk left it standing at. */
const onRelease = () => {
  if (!walking) return
  walking = false
  if (handed !== began) emit('commit', handed)
}

/**
 * The handle come to rest under the pointer. What it came to rest at is handed
 * on before the commit is emitted, so a caller acting on the second has
 * the first.
 */
const onCommit = (value: number[]) => {
  const next = value[0]
  if (typeof next !== 'number') return
  setValue(next)
  emit('commit', next)
}

const endDrag = () => {
  dragging = false
  window.removeEventListener('pointerup', endDrag)
  window.removeEventListener('pointercancel', endDrag)
}

/** The pointer goes down: the handle is the person's until the pointer is up. */
const onPointerDown = () => {
  if (props.isDisabled) return
  dragging = true
  window.addEventListener('pointerup', endDrag)
  window.addEventListener('pointercancel', endDrag)
}

onBeforeUnmount(endDrag)
</script>

<template>
  <SliderRoot
    data-slot="slider"
    orientation="horizontal"
    :model-value="[shown]"
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
