<script setup lang="ts">
/**
 * The fields a stencil names, edited: one row to a field, each carrying a
 * handle, the name typed in it and the way to be rid of it.
 *
 * The first field names the cards the stencil cuts, so it stands first and goes
 * nowhere. What is wrong with a field is the caller's, and stands under that
 * field's row.
 */
import { computed, useId } from 'vue'
import ErrorMessage from './ErrorMessage.vue'
import Icon from './Icon.vue'
import NameBox from './NameBox.vue'
import Divider from '../divider/Divider.vue'
import CardRow from './CardRow.vue'
import { useDrag } from './drag'
import { useNaming } from './naming'
import { Button } from '../components/ui/button'
import {
  landing,
  numbered,
  objection,
  directionOf,
  STEP_KEYS,
  type Problems,
  type InsertionPoint,
  type Objection,
} from './order'
import { fieldRows, STENCIL_WORDS, type StencilWords } from './stencil'

const props = withDefaults(
  defineProps<{
    /** The fields a card is asked for, each named once, in the order they stand. */
    fields: readonly string[]
    /** What the caller found wrong with each field, under the name it is declared by. */
    wrong?: Problems | null
    /** The words they are drawn with. */
    words?: StencilWords
  }>(),
  { wrong: null, words: () => STENCIL_WORDS },
)

const emit = defineEmits<{
  (event: 'add', name: string): void
  (event: 'rename', field: string, name: string): void
  (event: 'remove', field: string): void
  /** A field let go somewhere in the order: before another, or at the end. */
  (event: 'move', field: string, at: InsertionPoint): void
}>()

/** What these objections are named by, which is this list's alone. */
const uid = useId()

/** What is wrong with a name, where what it is wrong about says so. */
const objectsId = (over: string): string => `${uid}-${encodeURIComponent(over)}-objects`

/**
 * A name typed over the one a field carries. A field is named by its own name,
 * and what it is measured against is every other field's.
 */
const naming = useNaming<Objection>({
  carries: (field) => field,
  taken: (field) => props.fields.filter((each) => each !== field),
  amiss: objection,
  renamed: (field, name) => emit('rename', field, name),
})

/** Why what is in a field's box cannot be used, and nothing while it can. */
const objects = (field: string): Objection | null => naming.objection(field)

/**
 * The field under the pointer's hand, and where letting go would put it: before
 * a field, at the end, or nowhere. The first field names every card, so nothing
 * lands above it and it goes nowhere itself.
 */
const { dragged, at, lift, over, release, drop, step } = useDrag<InsertionPoint | undefined>({
  order: () => props.fields,
  nowhere: undefined,
  lands: (held, lands) => landing(props.fields, held, lands),
  moves: (held, lands) => emit('move', held, lands),
})

const rows = computed(() => fieldRows(props.fields, dragged.value))

/** What is said of a field's name that cannot be used, and nothing while it can. */
const says = (field: string): string | null => {
  const why = objects(field)
  return why === null ? null : props.words.objection(why)
}

/** What is wrong with one field, and nothing where nothing is. */
const wrongWith = (field: string): readonly string[] => props.wrong?.get(field) ?? []

const add = (): void => {
  emit('add', numbered(props.fields, props.words.fieldStem))
}

/** A field asked by the keyboard to go one place along the order. */
const onGripKey = (event: KeyboardEvent, field: string): void => {
  const direction = directionOf(event.key)
  if (direction !== null) step(field, direction, event)
}
</script>

<template>
  <!-- A field is let go anywhere in the part its list stands in, so the way the
       list is added by takes it and a stencil naming no field still has
       somewhere to put one. -->
  <section
    class="stencil__part"
    :aria-label="words.fields"
    @dragover="over(null, $event)"
    @drop="drop"
  >
    <h2 class="stencil__heading caps-numen m-0 text-small text-hushed">{{ words.fields }}</h2>

    <ul v-if="rows.length" class="stencil__fields">
      <li
        v-for="row in rows"
        :key="row.field"
        class="stencil__field caret-above"
        :data-field="row.field"
        :data-names="row.names || undefined"
        :data-dragged="row.dragged || undefined"
        :data-before="row.field === at || undefined"
        @dragover.stop="over(row.names ? undefined : row.field, $event)"
        @drop.stop="drop"
      >
        <CardRow class="stencil__row" :data-objects="objects(row.field) ?? undefined">
          <!-- The first field names every card, so its handle is there and
               turned off, and the row keeps the shape every other row has. The
               handle is what a row is dragged by, by the pointer and by the
               arrows along the order alike. -->
          <span
            class="stencil__grip flex shrink-0 items-center text-hushed"
            data-grip
            role="button"
            :tabindex="row.names ? -1 : 0"
            :draggable="!row.names"
            :data-disabled="row.names || undefined"
            :aria-disabled="row.names || undefined"
            :aria-label="row.names ? words.pinned : `${words.drag}: ${row.field}`"
            :aria-keyshortcuts="row.names ? undefined : STEP_KEYS"
            :title="row.names ? words.pinned : `${words.drag}: ${row.field}`"
            @dragstart="lift(row.field, $event)"
            @dragend="release"
            @keydown="onGripKey($event, row.field)"
          >
            <Icon shows="grip" />
          </span>

          <NameBox
            class="stencil__box min-w-0 flex-1"
            :naming="naming"
            :over="row.field"
            :stem="`${words.fieldStem} ${row.at}`"
            :described-by="objects(row.field) ? objectsId(row.field) : null"
          />

          <Button
            variant="ghost"
            size="icon-small"
            class="stencil__away size-6"
            :disabled="row.names"
            :title="row.names ? words.pinned : undefined"
            :aria-label="`${words.remove}: ${row.field}`"
            @click="emit('remove', row.field)"
          >
            <Icon shows="cross" />
          </Button>
        </CardRow>

        <ErrorMessage
          v-if="says(row.field)"
          :id="objectsId(row.field)"
          class="stencil__objects"
          role="alert"
          :said="says(row.field) ?? ''"
        />

        <ErrorMessage
          v-if="wrongWith(row.field).length"
          class="stencil__objects"
          data-wrong
          :said="wrongWith(row.field)"
          :label="words.wrong"
        />
      </li>
    </ul>

    <p v-else class="stencil__silence caps-numen m-0 text-small text-hushed">{{ words.noFields }}</p>

    <Divider>
      <Button variant="ghost" size="small" @click="add">
        <Icon shows="plus" />
        {{ words.addField }}
      </Button>
    </Divider>
  </section>
</template>

<style scoped>
@import './caret.css';

.stencil__fields {
  display: flex;
  flex-direction: column;
  /* The room between rows is the editor's, and this is what it comes to
     where the rows stand anywhere else. */
  gap: var(--row-gap, 0.5rem);
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
.stencil__row[data-objects] {
  border-color: var(--numen-alarm);
}

/* A press on the name works the box, and does not take hold of the row it
   stands in. */
.stencil__box {
  min-inline-size: 0;
}
</style>
