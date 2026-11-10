<script setup lang="ts">
/**
 * One part of the window a template is edited in: a half written, or what that half
 */
import { computed, useTemplateRef } from 'vue'
import { ErrorMessage } from '../../error-message'
import { AutosizeTextarea } from '../../autosize-textarea'
import { CardProse } from '../../card-prose'
import { CARD_MODEL_WORDS, type Pane, type CardModelWords } from '../../../../../lib/cards'

withDefaults(
  defineProps<{
    pane: Pane
    isFocused?: boolean
    words?: CardModelWords
  }>(),
  { isFocused: false, words: () => CARD_MODEL_WORDS },
)

const emit = defineEmits<{
  /** The text of the half after the change. */
  (event: 'write', text: string): void
  /** The box was typed in. */
  (event: 'aim'): void
}>()

defineExpose({
  /** The box this half is written in, for a caller that puts the caret in it. */
  box: computed<HTMLTextAreaElement | null>(() => textareaRef.value?.box ?? null),
})

const textareaRef = useTemplateRef<InstanceType<typeof AutosizeTextarea>>('written')
</script>

<template>
  <div
    class="template__pane"
    :data-pane="`${pane.half}-${pane.mode}`"
    :data-mode="pane.mode"
    :data-blank="pane.isBlank || undefined"
    :data-focused="isFocused || undefined"
  >
    <AutosizeTextarea
      v-if="pane.mode === 'written'"
      ref="written"
      class="template__box"
      :text="pane.text"
      :data-half="pane.half"
      :aria-label="pane.named"
      @focus="emit('aim')"
      @write="(text: string) => emit('write', text)"
    />

    <div
      v-else
      class="template__preview"
      :data-preview="pane.half"
      :aria-label="pane.named"
      role="group"
    >
      <!-- A preview is a template read, not a template followed: a link in it stays
           where it is pressed. -->
      <CardProse :text="pane.text" @follow="(_href, press) => press.preventDefault()" />
    </div>

    <!-- An empty part says what it is for, in the middle of itself. -->
    <p
      v-if="pane.isBlank"
      class="template__ghost caps-numen text-small text-hushed"
      aria-hidden="true"
    >
      {{ pane.label }}
    </p>

    <!-- What is wrong with the half is in the foot of the part, over what is
         written there. -->
    <div v-if="pane.stray.length" class="template__amiss">
      <ErrorMessage class="template__objections" role="alert" :said="words.stray(pane.stray)" />
    </div>
  </div>
</template>

<style scoped>
/* A part is a pane of the window: it has a background and no border of its own.
   What it holds and what it says while it holds nothing share its one cell. */
.template__pane {
  position: relative;
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
  min-inline-size: 0;
  min-block-size: var(--pane-min);
  background: var(--numen-field-bg);
}

.template__pane[data-mode='preview'] {
  background: var(--numen-raised);
}

.template__box,
.template__preview,
.template__ghost {
  grid-area: 1 / 1;
}

.template__amiss {
  --amiss-gap: 0.125rem;
  position: absolute;
  z-index: 1;
  inset-block-end: var(--box-air);
  inset-inline-end: var(--box-pad-inline);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--amiss-gap);
  max-inline-size: calc(100% - 2 * var(--box-pad-inline));
  pointer-events: none;
}

/* Problem messages are displayed over the content below them, so they have a background. */
.template__amiss > .template__objections {
  --objections-padding-block: 0.125rem;
  --objections-padding-inline: 0.375rem;
  padding: var(--objections-padding-block) var(--objections-padding-inline);
  border-radius: var(--numen-radius);
  background: var(--numen-alarm-bg);
}

/* A template's box starts at a few lines and grows with what is written in
   it, as every other box does. */
.template__box {
  --autosize-lines: var(--pane-lines);
}

.template__ghost {
  place-self: center;
  margin: 0;
  padding-inline: var(--box-pad-inline);
  text-align: center;
  pointer-events: none;
}

.template__preview {
  min-inline-size: 0;
  padding: var(--box-air) var(--box-pad-inline);
  overflow-wrap: anywhere;
}
</style>
