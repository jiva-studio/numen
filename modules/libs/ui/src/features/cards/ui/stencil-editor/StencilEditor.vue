<script setup lang="ts">
/**
 * A stencil, edited: the fields it names at the top and the faces that show
 * them under, the first field naming the cards this stencil cuts.
 *
 * What the stencil stands for is the caller's, and so is what is wrong with it:
 * a face's mark stands beside that face's name, a field's under that field's
 * row.
 */
import { computed } from 'vue'
import { Face } from './face'
import { Fields } from './fields'
import { AddFace } from './add-face'
import { useDrag } from '../../model/drag'
import { getDeclaredFields, getFreeName, type Half, type InsertionPoint } from '../../lib/order'
import {
  faceRows,
  NOTHING_AMISS,
  STENCIL_WORDS,
  type StencilFace,
  type StencilWords,
  type StencilWrong,
} from '../../lib/stencil'
import { sampleValues } from '../../lib/fill'

/**
 * What the caller found wrong with the stencil it handed in. A face's stands
 * beside that face's name and a field's under that field's row, so nothing is
 * said in a place that leaves a person guessing what it is about.
 */
const props = withDefaults(
  defineProps<{
    /** The fields, in the order a person is asked for them. The first names the card. */
    fields: readonly string[]
    /** The faces, in the order they are drawn. */
    faces: readonly StencilFace[]
    /** What the editor is announced as. */
    name?: string
    /** What the caller found wrong with the fields and the faces it handed in. */
    wrong?: StencilWrong
    /** The words it is drawn with. */
    words?: StencilWords
  }>(),
  {
    name: 'Stencil',
    wrong: () => NOTHING_AMISS,
    words: () => STENCIL_WORDS,
  },
)

const emit = defineEmits<{
  (event: 'add-field', name: string): void
  (event: 'rename-field', field: string, name: string): void
  (event: 'remove-field', field: string): void
  /** A field let go somewhere in the order: before another, or at the end. */
  (event: 'move-field', field: string, at: InsertionPoint): void
  (event: 'add-face', name: string): void
  (event: 'rename-face', id: string, name: string): void
  (event: 'remove-face', id: string): void
  /**
   * A face let go somewhere in the order: before another, or at the end. The
   * order of the faces is the order a card's repetitions are taken from it, so
   * nothing among them is fixed.
   */
  (event: 'move-face', id: string, at: InsertionPoint): void
  /** One half of one face, as it now reads. */
  (event: 'write', id: string, half: Half, text: string): void
}>()

/** The fields a card is asked for, a name declared twice naming one field. */
const asked = computed(() => getDeclaredFields(props.fields))

/**
 * The face under the pointer's hand, and where letting go would put it. The
 * order of the faces is the order a card's repetitions are taken from it, so
 * nothing among them is fixed and a face lands anywhere but where it stands.
 */
const {
  dragged: face,
  at: faceAt,
  lift: liftFace,
  hover: hoverFace,
  release: releaseFace,
  drop: dropFace,
  step: stepFace,
} = useDrag<InsertionPoint>({
  order: () => props.faces.map((each) => each.id),
  nowhere: null,
  isMoved: (held, lands) => lands !== held,
  move: (held, lands) => emit('move-face', held, lands),
})

/** What a preview stands in the slots, which is each field under its own name. */
const sample = computed(() => sampleValues(asked.value))

/** The faces as they are drawn. */
const drawn = computed(() => faceRows(props.faces, asked.value, sample.value))

/** What is wrong with one face, and nothing where nothing is. */
const wrongWithFace = (id: string): readonly string[] => props.wrong.at.get(id) ?? []

const addFace = (): void => {
  emit('add-face', getFreeName(props.faces.map((each) => each.name), props.words.faceStem))
}
</script>

<template>
  <div
    class="stencil numen bg-surface font-sans text-base text-ink"
    role="group"
    :aria-label="name"
  >
    <Fields
      :fields="asked"
      :wrong="wrong.fields"
      :words="words"
      @add="(name: string) => emit('add-field', name)"
      @rename="(field: string, name: string) => emit('rename-field', field, name)"
      @remove="(field: string) => emit('remove-field', field)"
      @move="(field: string, at: InsertionPoint) => emit('move-field', field, at)"
    />

    <section
      class="stencil__part"
      :aria-label="words.faces"
      @dragover="hoverFace(null, $event)"
      @drop="dropFace"
    >
      <h2 class="stencil__heading caps-numen m-0 text-small text-hushed">{{ words.faces }}</h2>

      <p v-if="!drawn.length" class="stencil__silence caps-numen m-0 text-small text-hushed">
        {{ words.noFaces }}
      </p>

      <Face
        v-for="one in drawn"
        :key="one.id"
        class="stencil__face caret-above"
        :face="one"
        :wrong="wrongWithFace(one.id)"
        :words="words"
        :data-dragged="one.id === face || undefined"
        :data-before="one.id === faceAt || undefined"
        @dragover.stop="hoverFace(one.id, $event)"
        @drop.stop="dropFace"
        @rename="(name: string) => emit('rename-face', one.id, name)"
        @remove="emit('remove-face', one.id)"
        @lift="liftFace(one.id, $event)"
        @release="releaseFace"
        @step="(direction, press) => stepFace(one.id, direction, press)"
        @write="(half, text) => emit('write', one.id, half, text)"
      />

      <AddFace :label="words.addFace" @press="addFace" />
    </section>
  </div>
</template>

<style scoped>
@import '../caret.css';

.stencil {
  /* The room between one part and the next, and between the rows of a part. */
  --part-gap: 1.5rem;
  --row-gap: 0.5rem;
  /* The caret stands in the middle of the room between two rows. */
  --caret-out: calc(-1 * var(--row-gap) / 2);

  display: flex;
  flex-direction: column;
  gap: var(--part-gap);
  padding: var(--numen-gutter);
  overflow: auto;
}

.stencil__part {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--row-gap);
}

.stencil__face[data-dragged] {
  opacity: 0.5;
}
</style>
