<script setup lang="ts">
import { computed } from 'vue'
import { ErrorMessage } from '../../error-message'
import { Icon } from '../../icon'
import { NameBox } from '../../name-box'
import { CardRow } from '../../card-row'
import { Button } from '@numen/ui'
import type { NameEntryState } from '../../../model/nameEntry'
import {
  parseStepKey,
  STEP_KEYS,
  type InsertionPoint,
  type Objection,
  type StepDirection,
} from '../../../lib/order'
import { CARD_MODEL_WORDS, type FieldRow, type CardModelWords } from '../../../lib/card-model'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** Field row data and positional state. */
    row: FieldRow
    /** Name entry state managing this field. */
    naming: NameEntryState<Objection>
    /** Unique element identifier for error messages. */
    objectionsId: string
    /** Validation error messages for the field. */
    wrong?: readonly string[]
    /** Whether the drop indicator is positioned above this row. */
    isDropAbove?: boolean
    /** Localization words for card model UI. */
    words?: CardModelWords
  }>(),
  { wrong: () => [], isDropAbove: false, words: () => CARD_MODEL_WORDS },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'drag-over', target: InsertionPoint | undefined, press: DragEvent): void
  (event: 'drop'): void
  (event: 'lift', press: DragEvent): void
  (event: 'release'): void
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
  (event: 'remove'): void
}>()

/* --------------------------------- State ---------------------------------- */
const objections = computed(() => props.naming.getObjection(props.row.field))

const says = computed(() => {
  const objection = objections.value
  return objection === null ? null : props.words.objection(objection)
})

/* -------------------------------- Handlers -------------------------------- */
function onGripKey(event: KeyboardEvent): void {
  const direction = parseStepKey(event.key)
  if (direction !== null) emit('step', direction, event)
}
</script>

<template>
  <li
    class="card-model__field caret-above"
    :data-field="row.field"
    :data-names="row.hasNames || undefined"
    :data-dragged="row.isDragged || undefined"
    :data-before="isDropAbove || undefined"
    @dragover.stop="emit('drag-over', row.hasNames ? undefined : row.field, $event)"
    @drop.stop="emit('drop')"
  >
    <CardRow class="card-model__row" :data-objections="objections ?? undefined">
      <span
        class="card-model__grip text-hushed flex shrink-0 items-center"
        data-grip
        role="button"
        :tabindex="row.hasNames ? -1 : 0"
        :draggable="!row.hasNames"
        :data-disabled="row.hasNames || undefined"
        :aria-disabled="row.hasNames || undefined"
        :aria-label="row.hasNames ? words.pinned : `${words.drag}: ${row.field}`"
        :aria-keyshortcuts="row.hasNames ? undefined : STEP_KEYS"
        :title="row.hasNames ? words.pinned : `${words.drag}: ${row.field}`"
        @dragstart="emit('lift', $event)"
        @dragend="emit('release')"
        @keydown="onGripKey"
      >
        <Icon name="grip" />
      </span>

      <NameBox
        class="card-model__box min-w-0 flex-1"
        :naming="naming"
        :target-id="row.field"
        :stem="`${words.fieldStem} ${row.at}`"
        :described-by="objections ? objectionsId : null"
      />

      <Button
        variant="ghost"
        size="icon-small"
        class="card-model__away size-6"
        :disabled="row.hasNames"
        :title="row.hasNames ? words.pinned : undefined"
        :aria-label="`${words.remove}: ${row.field}`"
        @click="emit('remove')"
      >
        <Icon name="cross" />
      </Button>
    </CardRow>

    <ErrorMessage
      v-if="says"
      :id="objectionsId"
      class="card-model__objections"
      role="alert"
      :said="says"
    />

    <ErrorMessage
      v-if="wrong.length"
      class="card-model__objections"
      data-wrong
      :said="wrong"
      :label="words.wrong"
    />
  </li>
</template>

<style scoped>
@import '../../caret.css';

/* Field layout container. */
.card-model__field {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.card-model__field[data-dragged] {
  opacity: 0.5;
}

.card-model__grip {
  cursor: grab;
  user-select: none;
  -webkit-user-select: none;
}

/* Disabled grip styling. */
.card-model__grip[data-disabled] {
  opacity: 0.4;
  cursor: default;
}

/* Controls are muted until hovered or focused. */
.card-model__grip,
.card-model__away {
  opacity: 0.45;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.card-model__field:hover .card-model__grip:not([data-disabled]),
.card-model__field:hover .card-model__away:not(:disabled),
.card-model__field:focus-within .card-model__grip:not([data-disabled]),
.card-model__field:focus-within .card-model__away:not(:disabled) {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .card-model__grip,
  .card-model__away {
    transition: none;
  }
}

/* Validation objection indicator. */
.card-model__row[data-objections] {
  border-color: var(--numen-alarm);
}

.card-model__box {
  min-inline-size: 0;
}
</style>
