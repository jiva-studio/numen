<script setup lang="ts">
/**
 * One face of a stencil, edited: a strip it is dragged by, holding its name and
 * one row of the fields, and under that one window divided into four parts.
 *
 * Each half is written in a box of its own, and beside it stands what that half
 * comes to. A field pressed in the strip is written into the half last typed
 * in. What the face stands for is the caller's, and so is what is wrong with it.
 */
import { computed, nextTick, shallowRef, type ComponentPublicInstance } from 'vue'
import { CardHeader } from '../../card-header'
import { RemoveButton } from '../../remove-button'
import FacePane from './FacePane.vue'
import FaceStrip from './FaceStrip.vue'
import type { Half, StepDirection } from '../../../lib/order'
import {
  getPanes,
  STENCIL_WORDS,
  type FaceRow,
  type Pane,
  type StencilWords,
} from '../../../lib/stencil'
import { insert } from '../../../lib/fill'

const props = withDefaults(
  defineProps<{
    /** The face, laid out against the fields the stencil declares. */
    face: FaceRow
    /** What the caller found wrong with this face, said beside its name. */
    wrong?: readonly string[]
    /** The words it is drawn with. */
    words?: StencilWords
  }>(),
  { wrong: () => [], words: () => STENCIL_WORDS },
)

const emit = defineEmits<{
  (event: 'rename', name: string): void
  (event: 'remove'): void
  /** The face taken up by the pointer, and let go again. */
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  /** The face asked to go one place along the order. */
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
  /** One half of it, as it now reads. */
  (event: 'write', half: Half, text: string): void
}>()

/**
 * The half last typed in, and nothing while neither has been. A box nothing has
 * been typed in holds no caret, so what half the caret is in is one thing and
 * where the caret stands in it is another.
 */
const aim = shallowRef<Half | null>(null)

/** The half a field is written into, which is the front until one is typed in. */
const focusedHalf = computed<Half>(() => aim.value ?? 'front')

/** The two boxes this face is written in, each held by the part drawing it. */
const front = shallowRef<InstanceType<typeof FacePane> | null>(null)
const back = shallowRef<InstanceType<typeof FacePane> | null>(null)

const boxOf = (half: Half): HTMLTextAreaElement | null =>
  (half === 'front' ? front.value : back.value)?.box ?? null

/** A part was drawn, or taken away. The part a half is written in holds its box. */
const setBox = (pane: Pane, element: Element | ComponentPublicInstance | null): void => {
  if (pane.mode !== 'written') return
  const box = element as InstanceType<typeof FacePane> | null
  if (pane.half === 'front') front.value = box
  else back.value = box
}

/** The four parts this face's window is divided into. */
const divided = computed<readonly Pane[]>(() => getPanes(props.face, props.words))

/** The part a field would be written into. */
const isFocused = (pane: Pane): boolean => pane.mode === 'written' && focusedHalf.value === pane.half

/**
 * A field written into the half the caret is in, where the caret stands, the caret
 * following it. A box nothing has been typed in holds no caret, so the field
 * goes after what is written there.
 */
const put = async (field: string): Promise<void> => {
  const half = focusedHalf.value
  const target = boxOf(half)
  if (!target) return

  const caret = (aim.value === null ? null : target.selectionStart) ?? target.value.length
  const done = insert(target.value, caret, field)
  emit('write', half, done.text)
  await nextTick()
  const again = boxOf(half)
  again?.focus()
  again?.setSelectionRange(done.caret, done.caret)
}
</script>

<template>
  <article class="face rounded-node bg-raised flex flex-col" :data-face="face.id">
    <CardHeader
      :drag="`${words.drag}: ${face.name}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
      <FaceStrip
        :face="face"
        :wrong="wrong"
        :words="words"
        @rename="(name) => emit('rename', name)"
        @put="put"
      />

      <template #actions>
        <RemoveButton :label="`${words.remove}: ${face.name}`" @press="emit('remove')" />
      </template>
    </CardHeader>

    <!-- One window divided into four: the parts share the lines between them,
         and the frame around them is the face's own. -->
    <div class="face__body">
      <FacePane
        v-for="pane in divided"
        :key="`${pane.half}-${pane.mode}`"
        :ref="(held) => setBox(pane, held)"
        :pane="pane"
        :is-focused="isFocused(pane)"
        :words="words"
        @aim="aim = pane.half"
        @write="(text: string) => emit('write', pane.half, text)"
      />
    </div>
  </article>
</template>

<style scoped>
/* The width of the face is what the window inside it is divided by. */
.face {
  /* The air a box keeps inside a part of the window, which is what a box put
     there inherits. */
  --box-air: 0.5rem;
  --box-pad-inline: var(--numen-box-air);

  /* How tall a part of the window stands before what is in it makes it taller. */
  --pane-lines: 6;
  --pane-min: calc(var(--pane-lines) * var(--numen-line-height) * 1em + 2 * var(--box-air));

  position: relative;
  container-type: inline-size;
  inline-size: 100%;
  border: var(--numen-stroke) solid var(--numen-rule);
  overflow: hidden;
}

/* The body is one window: the parts are divided by the lines they share, and
   the frame around them is the face's own. */
.face__body {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--numen-stroke);
  background: var(--numen-rule);
}

/* Two parts to a row from the width at which a part still holds a line of a
   face: the writing beside its preview, the front above the back. */
@container (min-width: 36rem) {
  .face__body {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
