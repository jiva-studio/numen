<script setup lang="ts">
/**
 * One face of a stencil, edited: a strip it is dragged by, holding its name and
 * one row of the fields, and under that one window divided into four parts.
 *
 * Each half is written in a box of its own, and beside it stands what that half
 * comes to. A field pressed in the strip is written into the half last typed
 * in. What the face stands for is the caller's, and so is what is wrong with it.
 */
import { computed, nextTick, shallowRef, useId, type ComponentPublicInstance } from 'vue'
import ErrorMessage from './ErrorMessage.vue'
import CardHeader from './CardHeader.vue'
import RemoveButton from './RemoveButton.vue'
import AutosizeTextarea from './AutosizeTextarea.vue'
import CardProse from './CardProse.vue'
import NameBox from './NameBox.vue'
import { useNaming } from './naming'
import { Button } from '../components/ui/button'
import { heading, type Half, type Refusal, type StepDirection } from './order'
import {
  panes,
  STENCIL_WORDS,
  type FaceRow,
  type Pane,
  type StencilWords,
} from './stencil'
import { insert } from './fill'

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

/** What this face's objection is named by, which is this face's alone. */
const uid = useId()

const objectsId = `${uid}-objects`

/** A name typed over the one this face carries, until it is committed. */
const naming = useNaming<Refusal>({
  carries: () => props.face.name,
  taken: () => props.face.taken,
  amiss: heading,
  renamed: (_over, name) => emit('rename', name),
})

/** Why what is in the name box cannot be used, and nothing while it can. */
const objects = computed(() => naming.objection(props.face.id))

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
const front = shallowRef<InstanceType<typeof AutosizeTextarea> | null>(null)
const back = shallowRef<InstanceType<typeof AutosizeTextarea> | null>(null)

const boxOf = (half: Half): HTMLTextAreaElement | null =>
  (half === 'front' ? front.value : back.value)?.box ?? null

/** A part was drawn, or taken away, with the box its half is written in. */
const holds = (half: Half, drawn: Element | ComponentPublicInstance | null): void => {
  const box = drawn as InstanceType<typeof AutosizeTextarea> | null
  if (half === 'front') front.value = box
  else back.value = box
}

/** The four parts this face's window is divided into. */
const divided = computed<readonly Pane[]>(() => panes(props.face, props.words))

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
  <article class="face flex flex-col rounded-node bg-raised" :data-face="face.id">
    <CardHeader
      :drag="`${words.drag}: ${face.name}`"
      @dragstart="emit('lift', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
      <div class="face__said">
        <div class="face__head flex items-center">
          <NameBox
            class="face__title rounded-node"
            :naming="naming"
            :over="face.id"
            :stem="`${words.faceStem} ${face.at}`"
            :described-by="says ? objectsId : null"
          />

          <!-- The fields are small quiet chips, as small quiet actions are
               drawn everywhere else here. What they are for is said to a
               reader by the group, and to everyone else by their look. -->
          <div class="face__slots flex" role="group" :aria-label="`${words.insert}: ${face.name}`">
            <Button
              v-for="field in face.fields"
              :key="field"
              variant="ghost"
              size="small"
              class="face__slot h-5 rounded-pill bg-bubble px-2 text-small"
              draggable="false"
              :data-insert="field"
              :aria-label="`${words.insert}: ${field}`"
              @click="put(field)"
              >{{ field }}</Button
            >
          </div>
        </div>

        <!-- What is wrong with the face stands at the end of the strip, over
             the window under it. -->
        <div v-if="says || wrong.length" class="face__amiss">
          <ErrorMessage
            v-if="says"
            :id="objectsId"
            class="face__objects"
            role="alert"
            :said="says"
          />

          <ErrorMessage
            v-if="wrong.length"
            class="face__objects"
            data-wrong
            :said="wrong"
            :label="words.wrong"
          />
        </div>
      </div>

      <template #actions>
        <RemoveButton :label="`${words.remove}: ${face.name}`" @press="emit('remove')" />
      </template>
    </CardHeader>

    <!-- One window divided into four: the parts share the lines between them,
         and the frame around them is the face's own. -->
    <div class="face__body">
      <div
        v-for="pane in divided"
        :key="`${pane.half}-${pane.shows}`"
        class="face__pane"
        :data-pane="`${pane.half}-${pane.shows}`"
        :data-shows="pane.shows"
        :data-blank="pane.blank || undefined"
        :data-aimed="aiming(pane) || undefined"
      >
        <AutosizeTextarea
          v-if="pane.shows === 'written'"
          :ref="(held) => holds(pane.half, held)"
          class="face__box"
          :text="pane.text"
          :data-half="pane.half"
          :aria-label="pane.named"
          @focus="aim = pane.half"
          @write="(text: string) => emit('write', pane.half, text)"
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
        <p v-if="pane.blank" class="face__ghost caps-numen text-small text-hushed" aria-hidden="true">
          {{ pane.said }}
        </p>

        <!-- What is wrong with the half stands in the foot of the part, over
             what is written there. -->
        <div v-if="pane.stray.length" class="face__amiss">
          <ErrorMessage class="face__objects" role="alert" :said="words.stray(pane.stray)" />
        </div>
      </div>
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

/* The strip is the face's own, so its name leads it and the fields follow. */
.face__said {
  position: relative;
  inline-size: 100%;
}

.face__head {
  inline-size: 100%;
  gap: var(--numen-inset);
}

/* The name is the heading of the face: the largest thing in the strip, and
   never squeezed by however many fields stand beside it. */
.face__title {
  flex: 0 1 12rem;
  min-inline-size: 5rem;
  font-weight: 500;
}

/* The fields are a group under the heading, not its equal. A long row scrolls
   inside the strip. */
.face__slots {
  flex: 1 1 auto;
  min-inline-size: 0;
  gap: 0.25rem;
  overflow-x: auto;
  scrollbar-width: none;
}

.face__slots::-webkit-scrollbar {
  display: none;
}

.face__slot:hover {
  background: color-mix(in oklab, var(--numen-bubble-bg), var(--numen-ink) 10%);
}

/* The ring is drawn inside the chip, so the row it scrolls in cannot clip it. */
.face__slot:focus-visible {
  box-shadow: none;
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

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
.face__pane[data-shows='preview'] {
  background: var(--numen-raised);
}

.face__box,
.face__preview,
.face__ghost {
  grid-area: 1 / 1;
}

/* What is wrong stands at the end of what it is wrong about and over it, taking
   no room from it. A press meant for what is underneath reaches it. */
.face__amiss {
  position: absolute;
  z-index: 1;
  inset-inline-end: var(--box-pad-inline);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.125rem;
  max-inline-size: calc(100% - 2 * var(--box-pad-inline));
  pointer-events: none;
}

/* The strip's stands under it, and a part's inside its foot. */
.face__said > .face__amiss {
  inset-block-start: 100%;
}

.face__pane > .face__amiss {
  inset-block-end: var(--box-air);
}

/* What is wrong is read over whatever it covers, so it carries a ground. */
.face__amiss > .face__objects {
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
