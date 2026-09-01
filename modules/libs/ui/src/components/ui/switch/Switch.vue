<script setup lang="ts">
/**
 * On or off, in one place. The track fills with the accent while it is on and
 * the thumb slides across it.
 *
 * It is announced as a switch and answers the space bar, which is the
 * primitive's own behaviour.
 */
import type { HTMLAttributes } from 'vue'
import { SwitchRoot, SwitchThumb } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { disabled: false },
)

const model = defineModel<boolean>({ default: false })
</script>

<template>
  <SwitchRoot
    v-model="model"
    data-slot="switch"
    :disabled="disabled"
    :class="
      cn(
        'inline-flex h-5 w-9 shrink-0 items-center rounded-pill p-px',
        'cursor-pointer transition-colors duration-100 ease-numen',
        'bg-hushed data-[state=checked]:bg-accent',
        'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  >
    <SwitchThumb
      :class="
        cn(
          'block size-4 rounded-pill bg-raised',
          'transition-transform duration-100 ease-numen',
          'data-[state=checked]:translate-x-4',
        )
      "
    />
  </SwitchRoot>
</template>
