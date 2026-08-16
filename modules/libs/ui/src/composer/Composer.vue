<script setup lang="ts">
/**
 * The field a message is written in, and the button that sends it.
 *
 * Grows with what is typed until it reaches the height the tokens allow, then
 * scrolls. It says what was written and leaves clearing it to whoever
 * answers, so a message that failed to go is still there to try again.
 *
 * While an answer is on its way the button gives its place to three dots.
 */
import { computed, useTemplateRef } from 'vue'
import { Textarea } from '@/components/ui/textarea'
import { Button, buttonVariants } from '@/components/ui/button'
import Dots from '@/dots/Dots.vue'
import { cn } from '@/lib/utils'
import { COMPOSER_STATES, composerState, keyIntent, said } from './model'

const props = withDefaults(
  defineProps<{
    placeholder?: string
    /** An answer is being written; the dots stand in for the button. */
    working?: boolean
    disabled?: boolean
  }>(),
  { placeholder: 'Write a message', working: false, disabled: false },
)

const emit = defineEmits<{
  /** Sent. Carries what was written, with the whitespace around it gone. */
  (event: 'submit', text: string): void
}>()

const text = defineModel<string>({ default: '' })

const field = useTemplateRef<InstanceType<typeof Textarea>>('field')

const state = computed(() => composerState(text.value, props.working))
const descriptor = computed(() => COMPOSER_STATES[state.value])

/** Nothing to say, an answer already on its way, or turned off. */
const barred = computed(() => props.disabled || !descriptor.value.acts)

const act = () => {
  if (barred.value) return
  emit('submit', said(text.value))
}

const onKeydown = (event: KeyboardEvent) => {
  if (keyIntent(event) !== 'submit') return
  event.preventDefault()
  act()
}

defineExpose({ focus: () => field.value?.focus() })
</script>

<template>
  <div
    class="composer numen rounded-field border border-field-rule bg-field font-sans text-base text-ink focus-within:ring-(length:--numen-ring-width) focus-within:ring-ring"
  >
    <!-- The field and a copy of what is in it share one grid cell. The copy
         is what has a height, so the field is as tall as its text without
         anything measuring it. The trailing space holds the last line open
         while it is empty. -->
    <div class="composer__grow grid min-w-0">
      <div class="composer__mirror">{{ text }}&nbsp;</div>
      <Textarea
        ref="field"
        v-model="text"
        rows="1"
        class="composer__field resize-none overflow-y-auto rounded-none border-0 bg-transparent p-0 focus-visible:ring-0"
        :placeholder="placeholder"
        :disabled="disabled"
        @keydown="onKeydown"
      />
    </div>

    <!-- Out of the flow, so what stands here never decides how tall a row of
         typing is. One disc either way; the glyph on it is what changes. -->
    <div class="composer__action">
      <Transition name="composer__swap" mode="out-in">
        <span
          v-if="descriptor.shows === 'writing'"
          key="writing"
          :class="cn(buttonVariants({ size: 'icon' }), 'cursor-default')"
        >
          <Dots />
        </span>
        <Button v-else key="send" size="icon" :disabled="barred" @click="act">
          <slot name="glyph">
            <svg
              viewBox="0 0 16 16"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M8 13V3" />
              <path d="M3.5 7.5 8 3l4.5 4.5" />
            </svg>
          </slot>
        </Button>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
/* One row tall at rest, with the trailing end kept clear for whatever stands
   there. The room is the same width for the button and for the dots. */
.composer {
  position: relative;
  display: flex;
  min-block-size: var(--numen-field-min);
  padding: var(--numen-field-padding);
  padding-inline-end: calc(var(--numen-action-size) + 2 * var(--numen-field-padding));
}

.composer__grow {
  flex: 1;
}

/* The field and its copy occupy one cell, and every property that decides
   where a line breaks is set on both from here.

   The padding fills the row with one line of typing, which is what puts the
   typing and the button on the same middle. */
.composer__grow > * {
  grid-area: 1 / 1;
  padding-block: calc((var(--numen-action-size) - var(--numen-line-height) * 1em) / 2);
  padding-inline: var(--numen-field-text-inset);
  font: inherit;
  line-height: var(--numen-line-height);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* The copy is what the height is taken from, so the ceiling belongs on it. */
.composer__mirror {
  visibility: hidden;
  overflow: hidden;
  max-block-size: calc(
    (var(--numen-field-lines) - 1) * var(--numen-line-height) * 1em + var(--numen-action-size)
  );
}

.composer__field {
  scrollbar-width: none;
}

.composer__field::-webkit-scrollbar {
  display: none;
}

.composer__action {
  position: absolute;
  inset-block-end: var(--numen-field-padding);
  inset-inline-end: var(--numen-field-padding);
  display: flex;
  block-size: var(--numen-action-size);
  align-items: center;
  justify-content: center;
}

/* One disc gives way to the other, both halves together taking as long as a
   hover. */
.composer__swap-enter-active,
.composer__swap-leave-active {
  transition:
    opacity var(--numen-motion-hover) var(--numen-easing),
    transform var(--numen-motion-hover) var(--numen-easing);
}

.composer__swap-enter-from,
.composer__swap-leave-to {
  opacity: 0;
  transform: scale(0.75);
}
</style>
