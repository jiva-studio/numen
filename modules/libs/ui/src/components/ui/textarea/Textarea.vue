<script setup lang="ts">
/**
 * A field for several lines of text.
 *
 * It is sized by whatever lays it out. Growing with what is typed is the
 * business of the thing around it, which knows how tall it is allowed to get.
 */
import { useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()

const model = defineModel<string>({ default: '' })

const element = useTemplateRef<HTMLTextAreaElement>('element')

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: () => element.value?.focus(),
})
</script>

<template>
  <textarea
    ref="element"
    v-model="model"
    data-slot="textarea"
    :class="
      cn(
        'w-full rounded-node border border-field-rule bg-field px-3 py-2',
        'font-sans text-base text-ink placeholder:text-hushed',
        'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
