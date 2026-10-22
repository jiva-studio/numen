<script setup lang="ts">
import { computed } from 'vue'
import { ErrorMessage } from '../../error-message'
import { Icon } from '../../icon'
import { NameBox } from '../../name-box'
import { CardRow } from '../../card-row'
import { Button } from '@/shared/ui/button'
import type { NameEntryState } from '../../../model/nameEntry'
import {
  parseStepKey,
  STEP_KEYS,
  type InsertionPoint,
  type Objection,
  type StepDirection,
} from '../../../lib/order'
import { STENCIL_WORDS, type FieldRow, type StencilWords } from '../../../lib/stencil'

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
    /** Localization words for stencil UI. */
    words?: StencilWords
  }>(),
  { wrong: () => [], isDropAbove: false, words: () => STENCIL_WORDS },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'drag-over', at: InsertionPoint | undefined, press: DragEvent): void
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
    class="stencil__field caret-above"
    :data-field="row.field"
    :data-names="row.hasNames || undefined"
    :data-dragged="row.isDragged || undefined"
    :data-before="isDropAbove || undefined"
    @dragover.stop="emit('drag-over', row.hasNames ? undefined : row.field, $event)"
    @drop.stop="emit('drop')"
  >
    <CardRow class="stencil__row" :data-objections="objections ?? undefined">
      <span
        class="stencil__grip text-hushed flex shrink-0 items-center"
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
        class="stencil__box min-w-0 flex-1"
        :naming="naming"
        :target-id="row.field"
        :stem="`${words.fieldStem} ${row.at}`"
        :described-by="objections ? objectionsId : null"
      />

      <Button
        variant="ghost"
        size="icon-small"
        class="stencil__away size-6"
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

/* Field layout container. */
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

/* Disabled grip styling. */
.stencil__grip[data-disabled] {
  opacity: 0.4;
  cursor: default;
}

/* Controls are muted until hovered or focused. */
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

/* Validation objection indicator. */
.stencil__row[data-objections] {
  border-color: var(--numen-alarm);
}

.stencil__box {
  min-inline-size: 0;
}
</style>
