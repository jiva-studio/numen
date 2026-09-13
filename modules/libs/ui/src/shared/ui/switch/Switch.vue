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
import { cn } from '@/shared/lib/classes'

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
        'rounded-pill inline-flex h-5 w-9 shrink-0 items-center p-px',
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
          'duration-hover ease-numen transition-transform',
          'data-[state=checked]:translate-x-4',
        )
      "
    />
  </SwitchRoot>
</template>
