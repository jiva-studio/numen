<script setup lang="ts">
/**
 * On or off, in one place. The track fills with the accent while it is on and
 * the thumb slides across it.
 *
 * It is announced as a switch and answers the space bar, which is the
 * primitive's own behaviour.
 *
 * It is controlled: a press only emits `update:modelValue`, and what is drawn follows the
 * `modelValue` the parent passes. While `isBusy` is set (a write is in flight) presses are
 * ignored and nothing is drawn differently, so the thumb moves once, when the value changes.
 */
import type { HTMLAttributes } from 'vue'
import { SwitchRoot, SwitchThumb } from 'reka-ui'
import { cn } from '@/shared/lib/dom/classes'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    isDisabled?: boolean
    isBusy?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { isDisabled: false, isBusy: false },
)

const emit = defineEmits<{ (e: 'update:modelValue', value: boolean): void }>()

function onUpdate(value: boolean): void {
  if (props.isBusy) return
  emit('update:modelValue', value)
}
</script>

<template>
  <SwitchRoot
    :model-value="props.modelValue"
    data-slot="switch"
    :disabled="props.isDisabled"
    :aria-busy="props.isBusy || undefined"
    @update:model-value="onUpdate"
    :class="
      cn(
        'rounded-pill inline-flex h-5 w-9 shrink-0 items-center p-0.5',
        'duration-hover ease-numen cursor-pointer transition-colors',
        'bg-hushed data-[state=checked]:bg-accent',
        'ring-numen outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  >
    <SwitchThumb
      :class="
        cn(
          'rounded-pill bg-raised block size-4',
          // The thumb moves by its margin, a layout change, not by a transform: a transform
          // transition ends with one stray frame drawn at the start in WebKitGTK, which showed as
          // the thumb going back and moving again.
          'duration-hover ease-numen transition-[margin-inline-start]',
          'data-[state=checked]:ms-4',
        )
      "
    />
  </SwitchRoot>
</template>
