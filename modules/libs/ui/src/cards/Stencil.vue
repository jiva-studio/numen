<script setup lang="ts">
/**
 * A stencil, edited: the fields it names, and the faces that show them.
 *
 * The fields stand at the top and the faces under them, the first field naming
 * the cards this stencil cuts. Each face is a strip it is carried by, holding
 * its name and one row of the fields, and under that one window divided into
 * four parts. What the stencil stands for is the caller's.
 */
import { computed, nextTick, shallowRef, useTemplateRef } from 'vue'
import Bar from './Bar.vue'
import Marks from './Marks.vue'
import Glyph from './Glyph.vue'
import Grown from './Grown.vue'
import Rule from './Rule.vue'
import Slab from './Slab.vue'
import { Button } from '../components/ui/button'
import {
  aimedAt,
  faceBlocks,
  fieldRows,
  freeName,
  landing,
  objection,
  panes,
  STENCIL_WORDS,
  type Aim,
  type Draft,
  type FaceBlock,
  type Filled,
  type Half,
  type Landing,
  type Pane,
  type Shown,
  type StencilWords,
} from './model'
import { insert, sampled } from './fill'

const props = withDefaults(
  defineProps<{
    /** The fields, in the order a person is asked for them. The first names the card. */
    fields: readonly string[]
    /** The faces, in the order they are drawn. */
    faces: readonly Shown[]
    /** What the editor is announced as. */
    name?: string
    /** What a preview stands in the slots. Each field under its own name by default. */
    sample?: readonly Filled[]
    /** The words it is drawn with. */
    words?: StencilWords
  }>(),
  {
    name: 'Stencil',
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

const box = useTemplateRef<HTMLElement>('box')

/** A name being typed over the one a field carries. */
const draft = shallowRef<Draft | null>(null)

/**
 * The field under the pointer's hand, and where letting go would put it: before
 * a field, at the end, or nowhere.
 */
const carried = shallowRef<string | null>(null)
const at = shallowRef<Landing | undefined>(undefined)

/** The face under the pointer's hand, and where letting go would put it. */
const face = shallowRef<string | null>(null)
const faceAt = shallowRef<Landing>(null)

/** The box a field would be written into. */
const aim = shallowRef<Aim | null>(null)

const sample = computed(() => props.sample ?? sampled(props.fields))

const rows = computed(() => fieldRows(props.fields, draft.value, carried.value))

const blocks = computed(() => faceBlocks(props.faces, props.fields, sample.value))

/** The half of one face a field would be written into. */
const aiming = (id: string): Half => aimedAt(aim.value, id)

/** The four parts one face's window is divided into. */
const divided = (block: FaceBlock): readonly Pane[] => panes(block, props.words)

/** The part a field would be written into. */
const aimed = (id: string, pane: Pane): boolean =>
  pane.shows === 'written' && aiming(id) === pane.half

const typing = (field: string, text: string): void => {
  draft.value = { over: field, text }
}

/** A typed name committed, and nothing where it objects or says what it said. */
const commit = (field: string): void => {
  const held = draft.value
  draft.value = null
  if (!held || held.over !== field) return

  const name = held.text.trim()
  if (name === field) return
  if (objection(name, props.fields.filter((each) => each !== field))) return
  emit('rename-field', field, name)
}

const onNameKey = (event: KeyboardEvent, field: string): void => {
  if (event.key === 'Enter') {
    event.preventDefault()
    commit(field)
    ;(event.currentTarget as HTMLInputElement).blur()
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    draft.value = null
  }
}

const addField = (): void => {
  emit('add-field', freeName(props.fields, props.words.fieldStem))
}

const addFace = (): void => {
  emit('add-face', freeName(props.faces.map((each) => each.name), props.words.faceStem))
}

const lift = (field: string, event: DragEvent): void => {
  carried.value = field
  event.dataTransfer?.setData('text/plain', field)
}

const over = (lands: Landing | undefined, event: DragEvent): void => {
  if (carried.value === null) return
  event.preventDefault()
  at.value = lands
}

const drop = (): void => {
  const held = carried.value
  const lands = at.value
  carried.value = null
  at.value = undefined
  if (held === null || lands === undefined) return
  if (landing(props.fields, held, lands)) emit('move-field', held, lands)
}

const liftFace = (id: string, event: DragEvent): void => {
  face.value = id
  event.dataTransfer?.setData('text/plain', id)
}

const overFace = (lands: Landing, event: DragEvent): void => {
  if (face.value === null) return
  event.preventDefault()
  faceAt.value = lands
}

const dropFace = (): void => {
  const held = face.value
  const lands = faceAt.value
  face.value = null
  faceAt.value = null
  if (held !== null && lands !== held) emit('move-face', held, lands)
}

/** The box one half of one face is written in. */
const halfBox = (id: string, half: Half): HTMLTextAreaElement | null =>
  [...(box.value?.querySelectorAll<HTMLTextAreaElement>('[data-face][data-half]') ?? [])].find(
    (each) => each.dataset['face'] === id && each.dataset['half'] === half,
  ) ?? null

/** A field written into the half aimed at, where the caret stands, the caret following it. */
const put = async (id: string, field: string): Promise<void> => {
  const half = aiming(id)
  const target = halfBox(id, half)
  if (!target) return

  const done = insert(target.value, target.selectionStart ?? target.value.length, field)
  emit('write', id, half, done.text)
  await nextTick()
  const again = halfBox(id, half)
  again?.focus()
  again?.setSelectionRange(done.caret, done.caret)
}
</script>

<template>
  <div
    ref="box"
    class="stencil numen bg-surface font-sans text-base text-ink"
    role="group"
    :aria-label="name"
  >
    <section class="stencil__part" :aria-label="words.fields">
      <h2 class="stencil__heading text-small text-hushed">{{ words.fields }}</h2>

      <ul v-if="rows.length" class="stencil__fields" @dragover="over(null, $event)" @drop="drop">
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
          <Slab class="stencil__row" :data-objects="row.objection ?? undefined">
            <!-- The first field names every card, so its handle is there and
                 turned off, and the row keeps the shape every other row has. -->
            <span
              class="stencil__grip flex shrink-0 items-center text-hushed"
              data-grip
              :draggable="!row.names"
              :data-disabled="row.names || undefined"
              :title="row.names ? words.pinned : `${words.carry}: ${row.field}`"
              @dragstart="lift(row.field, $event)"
              @dragend="drop"
            >
              <Glyph shows="grip" />
            </span>

            <input
              class="stencil__box min-w-0 flex-1"
              type="text"
              :value="row.text"
              :placeholder="`${words.fieldStem} ${row.at}`"
              :aria-label="`${words.fieldStem} ${row.at}`"
              :aria-invalid="row.objection !== null || undefined"
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

          <p v-if="row.objection" class="stencil__objects text-small text-alarm" role="alert">
            {{ words.objection(row.objection) }}
          </p>
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

      <article
        v-for="block in blocks"
        :key="block.id"
        class="stencil__face flex flex-col rounded-node bg-raised"
        :data-face-block="block.id"
        :data-carried="block.id === face || undefined"
        :data-before="block.id === faceAt || undefined"
        @dragover.stop="overFace(block.id, $event)"
        @drop.stop="dropFace"
      >
        <Bar
          :carry="`${words.carry}: ${block.name}`"
          @dragstart="liftFace(block.id, $event)"
          @dragend="dropFace"
        >
          <div class="stencil__face-head flex items-center">
            <input
              class="stencil__title min-w-0 rounded-node"
              type="text"
              :value="block.name"
              :placeholder="`${words.faceStem} ${block.at}`"
              :aria-label="`${words.faceStem} ${block.at}`"
              @change="emit('rename-face', block.id, ($event.target as HTMLInputElement).value)"
            />

            <!-- The fields are small quiet chips, as small quiet actions are
                 drawn everywhere else here. What they are for is said to a
                 reader by the group, and to everyone else by their look. -->
            <div
              class="stencil__slots flex"
              role="group"
              :aria-label="`${words.insert}: ${block.name}`"
            >
              <Button
                v-for="slot in fields"
                :key="slot"
                variant="ghost"
                size="small"
                class="stencil__slot h-5 rounded-pill bg-bubble px-2 text-small"
                draggable="false"
                :data-insert="slot"
                :aria-label="`${words.insert}: ${slot}`"
                @click="put(block.id, slot)"
                >{{ slot }}</Button
              >
            </div>
          </div>

          <template #deeds>
            <Button
              variant="ghost"
              size="icon-small"
              class="size-6"
              draggable="false"
              :aria-label="`${words.remove}: ${block.name}`"
              @click="emit('remove-face', block.id)"
            >
              <Glyph shows="bin" />
            </Button>
          </template>
        </Bar>

        <!-- One window divided into four: the parts share the lines between
             them, and the frame around them is the block's own. -->
        <div class="stencil__face-body">
          <div
            v-for="pane in divided(block)"
            :key="`${pane.half}-${pane.shows}`"
            class="stencil__pane"
            :data-pane="`${pane.half}-${pane.shows}`"
            :data-shows="pane.shows"
            :data-blank="pane.blank || undefined"
            :data-aimed="aimed(block.id, pane) || undefined"
          >
            <Grown v-if="pane.shows === 'written'" class="stencil__grown" :text="pane.text">
              <textarea
                class="stencil__written"
                :data-face="block.id"
                :data-half="pane.half"
                :aria-label="pane.named"
                rows="1"
                :value="pane.text"
                @focus="aim = { face: block.id, half: pane.half }"
                @input="
                  emit('write', block.id, pane.half, ($event.target as HTMLTextAreaElement).value)
                "
              ></textarea>
            </Grown>

            <div
              v-else
              class="stencil__preview"
              :data-preview="pane.half"
              :aria-label="pane.named"
              role="group"
            >
              <Marks :text="pane.text" />
            </div>

            <!-- An empty part says what it is for, in the middle of itself. -->
            <p v-if="pane.blank" class="stencil__ghost text-small text-hushed" aria-hidden="true">
              {{ pane.said }}
            </p>

            <p v-if="pane.stray.length" class="stencil__objects text-small text-alarm" role="alert">
              {{ words.stray(pane.stray) }}
            </p>
          </div>
        </div>
      </article>

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

  /* The air a box keeps inside a part of a face's window, which is what a box
     put there inherits. */
  --box-air: 0.5rem;
  --box-pad-inline: 0.625rem;

  /* How tall a part of that window stands before what is in it makes it taller. */
  --pane-lines: 6;
  --pane-min: calc(var(--pane-lines) * var(--numen-line-height) * 1em + 2 * var(--box-air));

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

.stencil__silence {
  margin: 0;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* The width of the block is what the window inside it is divided by. */
.stencil__face {
  position: relative;
  container-type: inline-size;
  inline-size: 100%;
  border: var(--numen-stroke) solid var(--numen-node-border);
  overflow: hidden;
}

/* The body is one window: the parts are divided by the lines they share, and
   the frame around them is the block's own. */
.stencil__face-body {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--numen-stroke);
  background: var(--numen-node-border);
}

/* Two parts to a row from the width at which a part still holds a line of a
   face: the writing beside its preview, the front above the back. */
@container (min-width: 36rem) {
  .stencil__face-body {
    grid-template-columns: 1fr 1fr;
  }
}

/* The strip is the face's own, so its name leads it and the fields follow. */
.stencil__face-head {
  inline-size: 100%;
  gap: var(--numen-inset);
}

/* The name is the heading of the block: the largest thing in the strip, and
   never squeezed by however many fields stand beside it. */
.stencil__title {
  flex: 0 1 12rem;
  min-inline-size: 5rem;
  font-weight: 500;
}

/* The fields are a group under the heading, not its equal. A long row scrolls
   inside the strip rather than pushing the name aside or growing the strip. */
.stencil__slots {
  flex: 1 1 auto;
  min-inline-size: 0;
  gap: 0.25rem;
  overflow-x: auto;
  scrollbar-width: none;
}

.stencil__slots::-webkit-scrollbar {
  display: none;
}

.stencil__slot:hover {
  background: color-mix(in oklab, var(--numen-bubble-bg), var(--numen-node-fg) 10%);
}

/* The ring is drawn inside the chip, so the row it scrolls in cannot clip it. */
.stencil__slot:focus-visible {
  box-shadow: none;
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

/* A part is a pane of the window: it carries a ground and no line of its own.
   What it holds and what it says while it holds nothing share its one cell. */
.stencil__pane {
  display: grid;
  grid-template-columns: 1fr;
  /* What the part holds takes the room; what is wrong takes a line under it. */
  grid-template-rows: 1fr;
  min-inline-size: 0;
  min-block-size: var(--pane-min);
  background: var(--numen-field-bg);
}

/* What is written stands on the ground a box stands on; what it comes to
   stands on the ground the block is read on. */
.stencil__pane[data-shows='preview'] {
  background: var(--numen-node-bg);
}

.stencil__grown,
.stencil__preview,
.stencil__ghost {
  grid-area: 1 / 1;
}

.stencil__pane > .stencil__objects {
  grid-area: 2 / 1;
  padding: 0 var(--box-pad-inline) var(--box-air);
}

/* A face's box stands open at a few lines and grows with what is written in
   it, as every other box does. */
.stencil__grown {
  --grown-lines: var(--pane-lines);
}

/* What an empty part is called stands in the middle of it, and is passed
   through to whatever is underneath. */
.stencil__ghost {
  place-self: center;
  margin: 0;
  padding-inline: var(--box-pad-inline);
  letter-spacing: var(--numen-caps-tracking);
  text-align: center;
  text-transform: uppercase;
  pointer-events: none;
}

.stencil__written {
  inline-size: 100%;
  min-inline-size: 0;
}

.stencil__preview {
  min-inline-size: 0;
  padding: var(--box-air) var(--box-pad-inline);
  overflow-wrap: anywhere;
}
</style>
