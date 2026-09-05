<script setup lang="ts">
/**
 * A button, in the sizes and colours the tokens name.
 *
 * `as` and `as-child` come from the primitive underneath: a link that looks
 * like a button is this component rendered as an anchor.
 */
import type { PrimitiveProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { Primitive } from 'reka-ui'
import { cn } from '@/classes'
import { buttonVariants, type ButtonVariants } from './variants'

const props = withDefaults(
  defineProps<
    PrimitiveProps & {
      variant?: ButtonVariants['variant']
      size?: ButtonVariants['size']
      class?: HTMLAttributes['class']
    }
  >(),
  { as: 'button', asChild: false },
)
</script>

<template>
  <Primitive
    data-slot="button"
    :as="as"
    :as-child="asChild"
    :class="cn(buttonVariants({ variant, size }), props.class)"
  >
    <slot />
  </Primitive>
</template>
