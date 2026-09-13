<script setup lang="ts">
/**
 * One field of a stencil, edited: a handle it is dragged by, the name typed in
 * it, and the way to be rid of it. What is wrong with it stands under its row.
 */
import { computed } from 'vue'
import { ErrorMessage } from '../../error-message'
import { Icon } from '../../icon'
import { NameBox } from '../../name-box'
import { CardRow } from '../../card-row'
import { Button } from '@/shared/ui/button'
import type { NamingState } from '../../../model/naming'
import {
  directionOf,
  STEP_KEYS,
  type InsertionPoint,
  type Objection,
  type StepDirection,
} from '../../../lib/order'
import { STENCIL_WORDS, type FieldRow, type StencilWords } from '../../../lib/stencil'

const props = withDefaults(
  defineProps<{
    /** The field, laid out against the order it stands in. */
    row: FieldRow
    /** The name typed over the one it carries, held by the list it stands in. */
    naming: NamingState<Objection>
    /** What is wrong with its name is named by this, which is this field's alone. */
    objectionsId: string
    /** What the caller found wrong with this field. */
    wrong?: readonly string[]
    /** A field let go here would land before it. */
    before?: boolean
    /** The words it is drawn with. */
    words?: StencilWords
  }>(),
  { wrong: () => [], before: false, words: () => STENCIL_WORDS },
)

const emit = defineEmits<{
  /** A field is dragged over this row. */
  (event: 'drag-over', at: InsertionPoint | undefined, press: DragEvent): void
  (event: 'drop'): void
  /** This field taken up by the pointer, and let go again. */
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  /** This field asked to go one place along the order. */
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
  (event: 'remove'): void
}>()

/** Why what is in the name box cannot be used, and nothing while it can. */
const objections = computed(() => props.naming.getObjection(props.row.field))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => {
  const objection = objections.value
  return objection === null ? null : props.words.objection(objection)
})

/** A field asked by the keyboard to go one place along the order. */
const onGripKey = (event: KeyboardEvent): void => {
  const direction = directionOf(event.key)
  if (direction !== null) emit('step', direction, event)
}
</script>

<template>
  <li
    class="stencil__field caret-above"
    :data-field="row.field"
    :data-names="row.names || undefined"
    :data-dragged="row.dragged || undefined"
    :data-before="before || undefined"
    @dragover.stop="emit('drag-over', row.names ? undefined : row.field, $event)"
    @drop.stop="emit('drop')"
  >
    <CardRow class="stencil__row" :data-objections="objections ?? undefined">
      <!-- The first field names every card, so its handle is there and
           turned off, and the row keeps the shape every other row has. The
           handle is what a row is dragged by, by the pointer and by the
           arrows along the order alike. -->
      <span
        class="stencil__grip text-hushed flex shrink-0 items-center"
        data-grip
        role="button"
        :tabindex="row.names ? -1 : 0"
        :draggable="!row.names"
        :data-disabled="row.names || undefined"
        :aria-disabled="row.names || undefined"
        :aria-label="row.names ? words.pinned : `${words.drag}: ${row.field}`"
        :aria-keyshortcuts="row.names ? undefined : STEP_KEYS"
        :title="row.names ? words.pinned : `${words.drag}: ${row.field}`"
        @dragstart="emit('lift', $event)"
        @dragend="emit('release')"
        @keydown="onGripKey"
      >
        <Icon name="grip" />
      </span>

      <NameBox
        class="stencil__box min-w-0 flex-1"
        :naming="naming"
        :over="row.field"
        :stem="`${words.fieldStem} ${row.at}`"
        :described-by="objections ? objectionsId : null"
      />

      <Button
        variant="ghost"
        size="icon-small"
        class="stencil__away size-6"
        :disabled="row.names"
        :title="row.names ? words.pinned : undefined"
        :aria-label="`${words.remove}: ${row.field}`"
        @click="emit('remove')"
      >
        <Icon name="cross" />
      </Button>
    </CardRow>

    <ErrorMessage
      v-if="says"
      :id="objectionsId"
      class="stencil__objections"
      role="alert"
      :said="says"
    />

    <ErrorMessage
      v-if="wrong.length"
      class="stencil__objections"
      data-wrong
      :said="wrong"
      :label="words.wrong"
    />
  </li>
</template>

<style scoped>
@import '../../caret.css';

/* The row is one block; what is wrong with it stands under that block. */
.stencil__field {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.stencil__field[data-dragged] {
  opacity: 0.5;
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
.stencil__row[data-objections] {
  border-color: var(--numen-alarm);
}

/* A press on the name works the box, and does not take hold of the row it
   stands in. */
.stencil__box {
  min-inline-size: 0;
}
</style>
