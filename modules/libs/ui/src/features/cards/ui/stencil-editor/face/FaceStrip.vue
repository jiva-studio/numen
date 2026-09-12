<script setup lang="ts">
/**
 * The strip a face is led by: its name, the row of fields that may be written
 * into it, and what is wrong with it.
 */
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { NameBox } from '../../name-box'
import { useNaming } from '../../../model/naming'
import { Button } from '@/shared/ui/button'
import { checkHeadingName, type HeadingObjection } from '../../../lib/order'
import { STENCIL_WORDS, type FaceRow, type StencilWords } from '../../../lib/stencil'

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
  /** A field was pressed, to be written into the face. */
  (event: 'put', field: string): void
}>()

/** What this face's objection is named by, which is this face's alone. */
const uid = useId()

const objectionsId = `${uid}-objections`

/** A name typed over the one this face carries, until it is committed. */
const naming = useNaming<HeadingObjection>({
  getName: () => props.face.name,
  getTakenNames: () => props.face.taken,
  checkName: checkHeadingName,
  rename: (_over, name) => emit('rename', name),
})

/** Why what is in the name box cannot be used, and nothing while it can. */
const objections = computed(() => naming.getObjection(props.face.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => {
  const objection = objections.value
  return objection === null ? null : props.words.faceObjection(objection)
})
</script>

<template>
  <div class="face__said">
    <div class="face__head flex items-center">
      <NameBox
        class="face__title rounded-node"
        :naming="naming"
        :over="face.id"
        :stem="`${words.faceStem} ${face.at}`"
        :described-by="says ? objectionsId : null"
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
          @click="emit('put', field)"
          >{{ field }}</Button
        >
      </div>
    </div>

    <!-- What is wrong with the face stands at the end of the strip, over
         the window under it. -->
    <div v-if="says || wrong.length" class="face__amiss">
      <ErrorMessage v-if="says" :id="objectionsId" class="face__objections" role="alert" :said="says" />

      <ErrorMessage
        v-if="wrong.length"
        class="face__objections"
        data-wrong
        :said="wrong"
        :label="words.wrong"
      />
    </div>
  </div>
</template>

<style scoped>
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
  outline: none;
}

/* What is wrong with the face stands under the strip and over the window,
   taking no room from it. A press meant for what is underneath reaches it. */
.face__amiss {
  position: absolute;
  z-index: 1;
  inset-block-start: 100%;
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
</style>
