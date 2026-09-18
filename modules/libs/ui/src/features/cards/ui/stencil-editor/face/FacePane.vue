<script setup lang="ts">
/**
 * One part of the window a face is edited in: a half written, or what that half
 * comes to. What is wrong with the half stands in its foot.
 */
import { computed, useTemplateRef } from 'vue'
import { ErrorMessage } from '../../error-message'
import { AutosizeTextarea } from '../../autosize-textarea'
import { CardProse } from '../../card-prose'
import { STENCIL_WORDS, type Pane, type StencilWords } from '../../../lib/stencil'

withDefaults(
  defineProps<{
    /** The part, as the face lays it out. */
    pane: Pane
    /** A field pressed in the strip would be written here. */
    isFocused?: boolean
    /** The words it is drawn with. */
    words?: StencilWords
  }>(),
  { isFocused: false, words: () => STENCIL_WORDS },
)

const emit = defineEmits<{
  /** The half as it now reads. */
  (event: 'write', text: string): void
  /** The box was typed in. */
  (event: 'aim'): void
}>()

const written = useTemplateRef<InstanceType<typeof AutosizeTextarea>>('written')

defineExpose({
  /** The box this half is written in, for a caller that puts the caret in it. */
  box: computed<HTMLTextAreaElement | null>(() => written.value?.box ?? null),
})
</script>

<template>
  <div
    class="face__pane"
    :data-pane="`${pane.half}-${pane.mode}`"
    :data-mode="pane.mode"
    :data-blank="pane.isBlank || undefined"
    :data-focused="isFocused || undefined"
  >
    <AutosizeTextarea
      v-if="pane.mode === 'written'"
      ref="written"
      class="face__box"
      :text="pane.text"
      :data-half="pane.half"
      :aria-label="pane.named"
      @focus="emit('aim')"
      @write="(text: string) => emit('write', text)"
    />

    <div
      v-else
      class="face__preview"
      :data-preview="pane.half"
      :aria-label="pane.named"
      role="group"
    >
      <!-- A preview is a face read, not a face followed: a link in it stays
           where it is pressed. -->
      <CardProse :text="pane.text" @follow="(_href, press) => press.preventDefault()" />
    </div>

    <!-- An empty part says what it is for, in the middle of itself. -->
    <p v-if="pane.isBlank" class="face__ghost caps-numen text-small text-hushed" aria-hidden="true">
      {{ pane.said }}
    </p>

    <!-- What is wrong with the half stands in the foot of the part, over what is
         written there. -->
    <div v-if="pane.stray.length" class="face__amiss">
      <ErrorMessage class="face__objections" role="alert" :said="words.stray(pane.stray)" />
    </div>
  </div>
</template>

<style scoped>
/* A part is a pane of the window: it carries a ground and no line of its own.
   What it holds and what it says while it holds nothing share its one cell. */
.face__pane {
  position: relative;
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
  min-inline-size: 0;
  min-block-size: var(--pane-min);
  background: var(--numen-field-bg);
}

/* What is written stands on the ground a box stands on; what it comes to
   stands on the ground the face is read on. */
.face__pane[data-mode='preview'] {
  background: var(--numen-raised);
}

.face__box,
.face__preview,
.face__ghost {
  grid-area: 1 / 1;
}

/* What is wrong stands in the foot of the part and over what is written there,
   taking no room from it. A press meant for what is underneath reaches it. */
.face__amiss {
  position: absolute;
  z-index: 1;
  inset-block-end: var(--box-air);
  inset-inline-end: var(--box-pad-inline);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.125rem;
  max-inline-size: calc(100% - 2 * var(--box-pad-inline));
  pointer-events: none;
}

/* What is wrong is read over whatever it covers, so it carries a ground. */
.face__amiss > .face__objections {
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius);
  background: var(--numen-alarm-bg);
}

/* A face's box stands open at a few lines and grows with what is written in
   it, as every other box does. */
.face__box {
  --autosize-lines: var(--pane-lines);
}

/* What an empty part is called stands in the middle of it, and is passed
   through to whatever is underneath. */
.face__ghost {
  place-self: center;
  margin: 0;
  padding-inline: var(--box-pad-inline);
  text-align: center;
  pointer-events: none;
}

.face__preview {
  min-inline-size: 0;
  padding: var(--box-air) var(--box-pad-inline);
  overflow-wrap: anywhere;
}
</style>
