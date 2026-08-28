<script setup lang="ts">
/**
 * A stencil, edited: the fields it names, and the faces that show them.
 *
 * The fields stand at the top and the faces under them, the first field naming
 * the cards this stencil cuts. Each face is a strip it is carried by, holding
 * its name and one row of the fields, and under that one window divided into
 * four parts. What the stencil stands for is the caller's, and so is what is
 * wrong with it: a face's mark stands under that face's name, a field's under
 * that field's row.
 */
import { computed, useId } from 'vue'
import Block from './Block.vue'
import Glyph from './Glyph.vue'
import Rule from '../rule/Rule.vue'
import Slab from './Slab.vue'
import { useCarry } from './carry'
import { useNaming } from './naming'
import { Button } from '../components/ui/button'
import {
  declared,
  faceBlocks,
  fieldRows,
  freeName,
  landing,
  objection,
  wayOf,
  NOTHING_AMISS,
  STENCIL_WORDS,
  type Half,
  type Landing,
  type Objection,
  type Shown,
  type StencilWords,
  type StencilWrong,
} from './model'
import { sampled } from './fill'

/**
 * What the caller found wrong with the stencil it handed in. A face's stands
 * under that face's name and a field's under that field's row, so nothing is
 * said in a place that leaves a person guessing what it is about.
 */
const props = withDefaults(
  defineProps<{
    /** The fields, in the order a person is asked for them. The first names the card. */
    fields: readonly string[]
    /** The faces, in the order they are drawn. */
    faces: readonly Shown[]
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
  (event: 'move-field', field: string, at: Landing): void
  (event: 'add-face', name: string): void
  (event: 'rename-face', id: string, name: string): void
  (event: 'remove-face', id: string): void
  /**
   * A face let go somewhere in the order: before another, or at the end. The
   * order of the faces is the order a card's repetitions are taken from it, so
   * nothing among them is fixed.
   */
  (event: 'move-face', id: string, at: Landing): void
  /** One half of one face, as it now reads. */
  (event: 'write', id: string, half: Half, text: string): void
}>()

/** What this editor's objections are named by, which is this editor's alone. */
const uid = useId()

/** What is wrong with a name, where what it is wrong about says so. */
const objectsId = (over: string): string => `${uid}-${encodeURIComponent(over)}-objects`

/** The fields a card is asked for, a name declared twice naming one field. */
const asked = computed(() => declared(props.fields))

/**
 * A name typed over the one a field carries. A field is named by its own name,
 * and what it is measured against is every other field's.
 */
const {
  text: fieldText,
  objection: fieldObjects,
  typing,
  commit,
  onKey: onNameKey,
} = useNaming<Objection>({
  carries: (field) => field,
  taken: (field) => asked.value.filter((each) => each !== field),
  amiss: objection,
  renamed: (field, name) => emit('rename-field', field, name),
})

/**
 * The field under the pointer's hand, and where letting go would put it: before
 * a field, at the end, or nowhere. The first field names every card, so nothing
 * lands above it and it goes nowhere itself.
 */
const { carried, at, lift, over, release, drop, step: stepField } = useCarry<Landing | undefined>({
  order: () => asked.value,
  nowhere: undefined,
  lands: (held, lands) => landing(asked.value, held, lands),
  moves: (held, lands) => emit('move-field', held, lands),
})

/**
 * The face under the pointer's hand, and where letting go would put it. The
 * order of the faces is the order a card's repetitions are taken from it, so
 * nothing among them is fixed and a face lands anywhere but where it stands.
 */
const {
  carried: face,
  at: faceAt,
  lift: liftFace,
  over: overFace,
  release: releaseFace,
  drop: dropFace,
  step: stepFace,
} = useCarry<Landing>({
  order: () => props.faces.map((each) => each.id),
  nowhere: null,
  lands: (held, lands) => lands !== held,
  moves: (held, lands) => emit('move-face', held, lands),
})

/** What a preview stands in the slots, which is each field under its own name. */
const sample = computed(() => sampled(asked.value))

const rows = computed(() => fieldRows(asked.value, carried.value))

/** The faces as they are drawn. */
const blocks = computed(() => faceBlocks(props.faces, asked.value, sample.value))

/** What is said of a field's name that cannot be used, and nothing while it can. */
const fieldSays = (field: string): string | null => {
  const why = fieldObjects(field)
  return why === null ? null : props.words.objection(why)
}

/** What is wrong with one field, and nothing where nothing is. */
const wrongWith = (field: string): readonly string[] => props.wrong.fields.get(field) ?? []

/** What is wrong with one face, and nothing where nothing is. */
const wrongWithFace = (id: string): readonly string[] => props.wrong.at.get(id) ?? []

/** The names the faces other than this one carry. */
const takenFrom = (id: string): readonly string[] =>
  props.faces.filter((each) => each.id !== id).map((each) => each.name)

const addField = (): void => {
  emit('add-field', freeName(asked.value, props.words.fieldStem))
}

const addFace = (): void => {
  emit('add-face', freeName(props.faces.map((each) => each.name), props.words.faceStem))
}

/** A field asked by the keyboard to go one place along the order. */
const onGripKey = (event: KeyboardEvent, field: string): void => {
  const way = wayOf(event.key)
  if (way !== null) stepField(field, way, event)
}

</script>

<template>
  <div
    class="stencil numen bg-surface font-sans text-base text-ink"
    role="group"
    :aria-label="name"
  >
    <!-- A field is let go anywhere in the part its list stands in, as a face
         is, so the way the list is added by takes it and a stencil naming no
         field still has somewhere to put one. -->
    <section
      class="stencil__part"
      :aria-label="words.fields"
      @dragover="over(null, $event)"
      @drop="drop"
    >
      <h2 class="stencil__heading text-small text-hushed">{{ words.fields }}</h2>

      <ul v-if="rows.length" class="stencil__fields">
        <li
          v-for="row in rows"
          :key="row.field"
          class="stencil__field"
          :data-field="row.field"
          :data-names="row.names || undefined"
          :data-carried="row.carried || undefined"
          :data-before="row.field === at || undefined"
          @dragover.stop="over(row.names ? undefined : row.field, $event)"
          @drop.stop="drop"
        >
          <Slab class="stencil__row" :data-objects="fieldObjects(row.field) ?? undefined">
            <!-- The first field names every card, so its handle is there and
                 turned off, and the row keeps the shape every other row has.
                 The handle is what a row is carried by, by the pointer and by
                 the arrows along the order alike. -->
            <span
              class="stencil__grip flex shrink-0 items-center text-hushed"
              data-grip
              role="button"
              :tabindex="row.names ? -1 : 0"
              :draggable="!row.names"
              :data-disabled="row.names || undefined"
              :aria-disabled="row.names || undefined"
              :aria-label="row.names ? words.pinned : `${words.carry}: ${row.field}`"
              :title="row.names ? words.pinned : `${words.carry}: ${row.field}`"
              @dragstart="lift(row.field, $event)"
              @dragend="release"
              @keydown="onGripKey($event, row.field)"
            >
              <Glyph shows="grip" />
            </span>

            <input
              class="stencil__box min-w-0 flex-1"
              type="text"
              :value="fieldText(row.field)"
              :placeholder="`${words.fieldStem} ${row.at}`"
              :aria-label="`${words.fieldStem} ${row.at}`"
              :aria-invalid="fieldObjects(row.field) !== null || undefined"
              :aria-describedby="fieldObjects(row.field) ? objectsId(row.field) : undefined"
              @input="typing(row.field, ($event.target as HTMLInputElement).value)"
              @change="commit(row.field)"
              @keydown="onNameKey($event, row.field)"
            />

            <Button
              variant="ghost"
              size="icon-small"
              class="stencil__away size-6"
              :disabled="row.names"
              :title="row.names ? words.pinned : undefined"
              :aria-label="`${words.remove}: ${row.field}`"
              @click="emit('remove-field', row.field)"
            >
              <Glyph shows="cross" />
            </Button>
          </Slab>

          <p
            v-if="fieldSays(row.field)"
            :id="objectsId(row.field)"
            class="stencil__objects text-small text-alarm"
            role="alert"
          >
            {{ fieldSays(row.field) }}
          </p>

          <ul
            v-if="wrongWith(row.field).length"
            class="stencil__objects text-small text-alarm"
            :aria-label="words.wrong"
            data-wrong
          >
            <li v-for="(text, said) in wrongWith(row.field)" :key="said">{{ text }}</li>
          </ul>
        </li>
      </ul>

      <p v-else class="stencil__silence text-small text-hushed">{{ words.noFields }}</p>

      <Rule>
        <Button variant="ghost" size="small" @click="addField">
          <Glyph shows="plus" />
          {{ words.addField }}
        </Button>
      </Rule>
    </section>

    <section
      class="stencil__part"
      :aria-label="words.faces"
      @dragover="overFace(null, $event)"
      @drop="dropFace"
    >
      <h2 class="stencil__heading text-small text-hushed">{{ words.faces }}</h2>

      <p v-if="!blocks.length" class="stencil__silence text-small text-hushed">
        {{ words.noFaces }}
      </p>

      <Block
        v-for="block in blocks"
        :key="block.id"
        class="stencil__face"
        :block="block"
        :fields="asked"
        :taken="takenFrom(block.id)"
        :wrong="wrongWithFace(block.id)"
        :words="words"
        :data-carried="block.id === face || undefined"
        :data-before="block.id === faceAt || undefined"
        @dragover.stop="overFace(block.id, $event)"
        @drop.stop="dropFace"
        @rename="(name: string) => emit('rename-face', block.id, name)"
        @remove="emit('remove-face', block.id)"
        @lift="liftFace(block.id, $event)"
        @release="releaseFace"
        @step="(way, press) => stepFace(block.id, way, press)"
        @write="(half, text) => emit('write', block.id, half, text)"
      />

      <Rule>
        <Button variant="ghost" size="small" @click="addFace">
          <Glyph shows="plus" />
          {{ words.addFace }}
        </Button>
      </Rule>
    </section>
  </div>
</template>

<style scoped>
.stencil {
  /* The room between one block and the next, and between the rows of a block.
     The line a carried field would land on. */
  --part-gap: 1.5rem;
  --row-gap: 0.5rem;
  --caret: 2px;

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

/* A heading over what it names is small print, as it is everywhere else. */
.stencil__heading {
  margin: 0;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.stencil__fields {
  display: flex;
  flex-direction: column;
  gap: var(--row-gap);
  inline-size: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* The row is one block; what is wrong with it stands under that block. */
.stencil__field {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.stencil__field[data-carried],
.stencil__face[data-carried] {
  opacity: 0.5;
}

.stencil__field[data-before]::before,
.stencil__face[data-before]::before {
  content: '';
  position: absolute;
  inset-block-start: calc(-1 * var(--row-gap) / 2);
  inset-inline: 0;
  block-size: var(--caret);
  background: var(--numen-ring);
}

.stencil__grip {
  cursor: grab;
  user-select: none;
  -webkit-user-select: none;
}

/* What is turned off is quiet, and answers neither the pointer nor a gesture. */
.stencil__grip[data-disabled] {
  opacity: 0.4;
  cursor: default;
}

/* What a row is reached for stands quietly until the row is. */
.stencil__grip,
.stencil__away {
  opacity: 0.45;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.stencil__field:hover .stencil__grip:not([data-disabled]),
.stencil__field:hover .stencil__away:not(:disabled),
.stencil__field:focus-within .stencil__grip:not([data-disabled]),
.stencil__field:focus-within .stencil__away:not(:disabled) {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .stencil__grip,
  .stencil__away {
    transition: none;
  }
}

/* The line and the ground are the row's, so what is wrong is said by the row. */
.stencil__row[data-objects] {
  border-color: var(--numen-alarm);
}

/* What is wrong takes the whole row under what it is wrong about. */
.stencil__objects {
  flex-basis: 100%;
  margin: 0;
}

/* What the caller found wrong is a list, however many things it found. */
ul.stencil__objects {
  padding-inline-start: 1.1rem;
  list-style: disc;
  overflow-wrap: anywhere;
}

.stencil__silence {
  margin: 0;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* A name is typed in the row it stands in, and carries neither a line nor a
   ground of its own. A press on it works it, and does not take hold of the row
   it stands in. */
.stencil__box {
  min-inline-size: 0;
  padding: 0.125rem 0.375rem;
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  cursor: auto;
}

.stencil__box:focus-visible {
  outline: none;
}
</style>
