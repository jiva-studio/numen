<script setup lang="ts">
/**
 * A field for several lines of text.
 *
 * It is sized by its parent layout. Growing with the typed text is the
 * responsibility of the surrounding component, which defines the maximum height.
 */
import { useId, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@numen/ui'

const props = defineProps<{
  id?: string
  class?: HTMLAttributes['class']
}>()

const model = defineModel<string>({ default: '' })

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  get element() {
    return element.value
  },
  focus: (how?: Parameters<HTMLElement['focus']>[0]) => element.value?.focus(how),
})

const ownId = useId()

const element = useTemplateRef<HTMLTextAreaElement>('element')
</script>

<template>
  <textarea
    :id="id ?? ownId"
    ref="element"
    v-model="model"
    data-slot="textarea"
    :class="
      cn(
        'rounded-node border-field-rule bg-field w-full border px-3 py-2',
        'text-ink placeholder:text-hushed font-sans text-base',
        'ring-numen outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
