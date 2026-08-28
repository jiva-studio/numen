<script setup lang="ts">
/**
 * One face of a stencil, edited: a strip it is carried by, holding its name and
 * one row of the fields, and under that one window divided into four parts.
 *
 * Each half is written in a box of its own, and beside it stands what that half
 * comes to. A field pressed in the strip is written into the half last typed
 * in. What the face stands for is the caller's, and so is what is wrong with it.
 */
import { computed, nextTick, shallowRef, useId, type ComponentPublicInstance } from 'vue'
import Bar from './Bar.vue'
import Deed from './Deed.vue'
import Grown from './Grown.vue'
import Marks from './Marks.vue'
import { useNaming } from './naming'
import { Button } from '../components/ui/button'
import {
  heading,
  panes,
  STENCIL_WORDS,
  type Amiss,
  type FaceBlock,
  type Half,
  type Pane,
  type StencilWords,
  type Way,
} from './model'
import { insert } from './fill'

const props = withDefaults(
  defineProps<{
    /** The face, laid out against the fields the stencil declares. */
    block: FaceBlock
    /** The fields a card is asked for, which are what may be written into a half. */
    fields: readonly string[]
    /** The names the other faces carry, which this one's is measured against. */
    taken: readonly string[]
    /** What the caller found wrong with this face, said under its name. */
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
  (event: 'step', way: Way, press: KeyboardEvent): void
  /** One half of it, as it now reads. */
  (event: 'write', half: Half, text: string): void
}>()

/** What this block's objection is named by, which is this block's alone. */
const uid = useId()

const objectsId = `${uid}-objects`

/** A name typed over the one this face carries, until it is committed. */
const naming = useNaming<Amiss>({
  carries: () => props.block.name,
  taken: () => props.taken,
  amiss: heading,
  renamed: (_over, name) => emit('rename', name),
})

/** What is in the name box: the name it carries, or what is being typed over it. */
const text = computed(() => naming.text(props.block.id))

/** Why what is in the name box cannot be used, and nothing while it can. */
const objects = computed(() => naming.objection(props.block.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => {
  const why = objects.value
  return why === null ? null : props.words.faceObjection(why)
})

/**
 * The half last typed in, and nothing while neither has been. A box nothing has
 * been typed in holds no caret, so what half it is aimed at is one thing and
 * where the caret stands in it is another.
 */
const aim = shallowRef<Half | null>(null)

/** The half a field is written into, which is the front until one is typed in. */
const aimed = computed<Half>(() => aim.value ?? 'front')

/** The two boxes this face is written in, each held by the part drawing it. */
const front = shallowRef<InstanceType<typeof Grown> | null>(null)
const back = shallowRef<InstanceType<typeof Grown> | null>(null)

const boxOf = (half: Half): HTMLTextAreaElement | null =>
  (half === 'front' ? front.value : back.value)?.box ?? null

/** A part was drawn, or taken away, with the box its half is written in. */
const holds = (half: Half, drawn: Element | ComponentPublicInstance | null): void => {
  const grown = drawn as InstanceType<typeof Grown> | null
  if (half === 'front') front.value = grown
  else back.value = grown
}

/** The four parts this face's window is divided into. */
const divided = computed<readonly Pane[]>(() => panes(props.block, props.words))

/** The part a field would be written into. */
const aiming = (pane: Pane): boolean => pane.shows === 'written' && aimed.value === pane.half

/**
 * A field written into the half aimed at, where the caret stands, the caret
 * following it. A box nothing has been typed in holds no caret, so the field
 * goes after what is written there.
 */
const put = async (field: string): Promise<void> => {
  const half = aimed.value
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
  <article class="block flex flex-col rounded-node bg-raised" :data-face-block="block.id">
    <Bar
      :carry="`${words.carry}: ${block.name}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(way, press) => emit('step', way, press)"
    >
      <div class="block__said flex flex-col">
        <div class="block__head flex items-center">
          <input
            class="block__title min-w-0 rounded-node"
            type="text"
            :value="text"
            :placeholder="`${words.faceStem} ${block.at}`"
            :aria-label="`${words.faceStem} ${block.at}`"
            :aria-invalid="objects !== null || undefined"
            :aria-describedby="says ? objectsId : undefined"
            @input="naming.typing(block.id, ($event.target as HTMLInputElement).value)"
            @change="naming.commit(block.id)"
            @keydown="naming.onKey($event, block.id)"
          />

          <!-- The fields are small quiet chips, as small quiet actions are
               drawn everywhere else here. What they are for is said to a
               reader by the group, and to everyone else by their look. -->
          <div class="block__slots flex" role="group" :aria-label="`${words.insert}: ${block.name}`">
            <Button
              v-for="field in fields"
              :key="field"
              variant="ghost"
              size="small"
              class="block__slot h-5 rounded-pill bg-bubble px-2 text-small"
              draggable="false"
              :data-insert="field"
              :aria-label="`${words.insert}: ${field}`"
              @click="put(field)"
              >{{ field }}</Button
            >
          </div>
        </div>

        <p v-if="says" :id="objectsId" class="block__objects text-small text-alarm" role="alert">
          {{ says }}
        </p>

        <ul
          v-if="wrong.length"
          class="block__objects text-small text-alarm"
          :aria-label="words.wrong"
          data-wrong
        >
          <li v-for="(said, at) in wrong" :key="at">{{ said }}</li>
        </ul>
      </div>

      <template #deeds>
        <Deed shows="bin" :label="`${words.remove}: ${block.name}`" @press="emit('remove')" />
      </template>
    </Bar>

    <!-- One window divided into four: the parts share the lines between them,
         and the frame around them is the block's own. -->
    <div class="block__body">
      <div
        v-for="pane in divided"
        :key="`${pane.half}-${pane.shows}`"
        class="block__pane"
        :data-pane="`${pane.half}-${pane.shows}`"
        :data-shows="pane.shows"
        :data-blank="pane.blank || undefined"
        :data-aimed="aiming(pane) || undefined"
      >
        <Grown
          v-if="pane.shows === 'written'"
          :ref="(held) => holds(pane.half, held)"
          class="block__grown"
          :text="pane.text"
          :data-half="pane.half"
          :aria-label="pane.named"
          @focus="aim = pane.half"
          @write="(text: string) => emit('write', pane.half, text)"
        />

        <div
          v-else
          class="block__preview"
          :data-preview="pane.half"
          :aria-label="pane.named"
          role="group"
        >
          <!-- A preview is a face read, not a face followed: a link in it stays
               where it is pressed. -->
          <Marks :text="pane.text" @follow="(_href, press) => press.preventDefault()" />
        </div>

        <!-- An empty part says what it is for, in the middle of itself. -->
        <p v-if="pane.blank" class="block__ghost text-small text-hushed" aria-hidden="true">
          {{ pane.said }}
        </p>

        <p v-if="pane.stray.length" class="block__objects text-small text-alarm" role="alert">
          {{ words.stray(pane.stray) }}
        </p>
      </div>
    </div>
  </article>
</template>

<style scoped>
/* The width of the block is what the window inside it is divided by. */
.block {
  /* The air a box keeps inside a part of the window, which is what a box put
     there inherits. */
  --box-air: 0.5rem;
  --box-pad-inline: 0.625rem;

  /* How tall a part of the window stands before what is in it makes it taller. */
  --pane-lines: 6;
  --pane-min: calc(var(--pane-lines) * var(--numen-line-height) * 1em + 2 * var(--box-air));

  position: relative;
  container-type: inline-size;
  inline-size: 100%;
  border: var(--numen-stroke) solid var(--numen-node-border);
  overflow: hidden;
}

/* The body is one window: the parts are divided by the lines they share, and
   the frame around them is the block's own. */
.block__body {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--numen-stroke);
  background: var(--numen-node-border);
}

/* Two parts to a row from the width at which a part still holds a line of a
   face: the writing beside its preview, the front above the back. */
@container (min-width: 36rem) {
  .block__body {
    grid-template-columns: 1fr 1fr;
  }
}

/* The strip is the face's own, so its name leads it and the fields follow.
   What is wrong with the name stands under the row it is wrong about. */
.block__said {
  inline-size: 100%;
  gap: 0.125rem;
}

.block__head {
  inline-size: 100%;
  gap: var(--numen-inset);
}

/* A name is typed in the strip it stands in, and carries neither a line nor a
   ground of its own. A press on it works it, and does not take hold of what it
   stands in. It is the heading of the block: the largest thing in the strip,
   and never squeezed by however many fields stand beside it. */
.block__title {
  flex: 0 1 12rem;
  min-inline-size: 5rem;
  padding: 0.125rem 0.375rem;
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  font-weight: 500;
  cursor: auto;
}

.block__title:focus-visible {
  outline: none;
}

/* The fields are a group under the heading, not its equal. A long row scrolls
   inside the strip. */
.block__slots {
  flex: 1 1 auto;
  min-inline-size: 0;
  gap: 0.25rem;
  overflow-x: auto;
  scrollbar-width: none;
}

.block__slots::-webkit-scrollbar {
  display: none;
}

.block__slot:hover {
  background: color-mix(in oklab, var(--numen-bubble-bg), var(--numen-node-fg) 10%);
}

/* The ring is drawn inside the chip, so the row it scrolls in cannot clip it. */
.block__slot:focus-visible {
  box-shadow: none;
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

/* A part is a pane of the window: it carries a ground and no line of its own.
   What it holds and what it says while it holds nothing share its one cell. */
.block__pane {
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
.block__pane[data-shows='preview'] {
  background: var(--numen-node-bg);
}

.block__grown,
.block__preview,
.block__ghost {
  grid-area: 1 / 1;
}

/* What is wrong takes the whole row under what it is wrong about. */
.block__objects {
  flex-basis: 100%;
  margin: 0;
}

/* What the caller found wrong is a list, however many things it found. */
ul.block__objects {
  padding-inline-start: 1.1rem;
  list-style: disc;
  overflow-wrap: anywhere;
}

.block__pane > .block__objects {
  grid-area: 2 / 1;
  padding: 0 var(--box-pad-inline) var(--box-air);
}

/* A face's box stands open at a few lines and grows with what is written in
   it, as every other box does. */
.block__grown {
  --grown-lines: var(--pane-lines);
}

/* What an empty part is called stands in the middle of it, and is passed
   through to whatever is underneath. */
.block__ghost {
  place-self: center;
  margin: 0;
  padding-inline: var(--box-pad-inline);
  letter-spacing: var(--numen-caps-tracking);
  text-align: center;
  text-transform: uppercase;
  pointer-events: none;
}

.block__preview {
  min-inline-size: 0;
  padding: var(--box-air) var(--box-pad-inline);
  overflow-wrap: anywhere;
}
</style>
