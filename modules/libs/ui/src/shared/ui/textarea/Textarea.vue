<script setup lang="ts">
/**
 * A field for several lines of text.
 *
 * It is sized by whatever lays it out. Growing with what is typed is the
 * business of the thing around it, which knows how tall it is allowed to get.
 */
import { useId, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/shared/lib/dom/classes'

const props = defineProps<{
  /** What a `<label for>` names the field by. Without one the field makes up its own. */
  id?: string
  class?: HTMLAttributes['class']
}>()

const model = defineModel<string>({ default: '' })

const ownId = useId()

const element = useTemplateRef<HTMLTextAreaElement>('element')

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})
</script>

<template>
  <textarea
    ref="element"
    :id="id ?? ownId"
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
