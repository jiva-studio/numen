<script setup lang="ts">
/**
 * The field a message is written in, and the disc that sends it. It grows with
 * what is typed until it reaches the height the tokens allow, then scrolls.
 *
 * It says what was written and leaves clearing it to whoever answers. While an
 * answer is on its way the disc stops it.
 */
import { computed, useTemplateRef } from 'vue'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import { COMPOSER_STATES, composerState, keyIntent, said } from './state'

const props = withDefaults(
  defineProps<{
    placeholder?: string
    /** An answer is being written; the disc stops it. */
    working?: boolean
    disabled?: boolean
    /** What the disc is called while it sends. */
    sends?: string
    /** What the disc is called while it stops. */
    stops?: string
  }>(),
  {
    placeholder: 'Write a message',
    working: false,
    disabled: false,
    sends: 'Send',
    stops: 'Stop',
  },
)

const emit = defineEmits<{
  /** Sent. Carries what was written, with the whitespace around it gone. */
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
}>()

const text = defineModel<string>({ default: '' })

const field = useTemplateRef<InstanceType<typeof Textarea>>('field')

const state = computed(() => composerState(text.value, props.working))
const descriptor = computed(() => COMPOSER_STATES[state.value])

/** Nothing to say, or turned off. */
const barred = computed(() => props.disabled || !descriptor.value.acts)

/** What the disc is called. */
const named = computed(() => (descriptor.value.shows === 'stop' ? props.stops : props.sends))

const act = () => {
  if (barred.value) return
  if (descriptor.value.shows === 'stop') emit('stop')
  else emit('submit', said(text.value))
}

/** Enter sends, and while an answer is on its way it does nothing. */
const onKeydown = (event: KeyboardEvent) => {
  if (keyIntent(event) !== 'submit') return
  event.preventDefault()
  if (descriptor.value.shows === 'send') act()
}

defineExpose({ focus: (how?: FocusOptions) => field.value?.focus(how) })
</script>

<template>
  <div
    class="composer numen rounded-field border border-panel-rule bg-panel shadow-panel backdrop-blur-panel font-sans text-base text-ink focus-within:ring-(length:--numen-ring-width) focus-within:ring-ring"
  >
    <!-- The field, a copy of what is in it, and the words standing in for what
         is not typed, all in one grid cell. The copy is what has a height, so
         the field is as tall as its text without anything measuring it, and
         the trailing space holds the last line open while it is empty. -->
    <div class="composer__grow grid min-w-0">
      <div class="composer__mirror">{{ text }}&nbsp;</div>
      <div v-if="!text" class="composer__placeholder text-hushed" aria-hidden="true">
        {{ placeholder }}
      </div>
      <Textarea
        ref="field"
        v-model="text"
        rows="1"
        class="composer__field resize-none overflow-y-auto rounded-none border-0 bg-transparent p-0 placeholder:text-transparent"
        :placeholder="placeholder"
        :disabled="disabled"
        @keydown="onKeydown"
      />
    </div>

    <!-- Out of the flow, so what stands here never decides how tall a row of
         typing is. One disc either way; the glyph on it is what changes. -->
    <div class="composer__action">
      <Transition name="composer__swap" mode="out-in">
        <Button
          :key="descriptor.shows"
          size="icon"
          :disabled="barred"
          :aria-label="named"
          @click="act"
        >
          <slot v-if="descriptor.shows === 'send'" name="glyph">
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
          <svg v-else viewBox="0 0 16 16" class="size-4" fill="currentColor">
            <rect x="3" y="3" width="10" height="10" rx="2" />
          </svg>
        </Button>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
/* One row tall at rest, with the trailing end kept clear for the disc that
   stands there. */
.composer {
  /* Lines of typing it grows to before it scrolls. */
  --lines: 10;

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
    (var(--lines) - 1) * var(--numen-line-height) * 1em + var(--numen-action-size)
  );
}

/* One line, then an ellipsis, so a field at rest is one row at every width.
   The field carries the same words for a screen reader to announce. */
.composer__placeholder {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  pointer-events: none;
}

.composer__field {
  scrollbar-width: none;
}

/* The ring belongs to the pill around the field. */
.composer__field:focus-visible {
  box-shadow: none;
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
