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
import { Icon } from '../../icon'
import { Divider } from '../../divider'
import Field from './Field.vue'
import { useDrag } from '../../../model/drag'
import { useNaming } from '../../../model/naming'
import { Button } from '@/shared/ui/button'
import {
  landing,
  getFreeName,
  objection,
  type Problems,
  type InsertionPoint,
  type Objection,
} from '../../../lib/order'
import { fieldRows, STENCIL_WORDS, type StencilWords } from '../../../lib/stencil'

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

/**
 * The field under the pointer's hand, and where letting go would put it: before
 * a field, at the end, or nowhere. The first field names every card, so nothing
 * lands above it and it goes nowhere itself.
 */
const { dragged, at, lift, over, release, drop, step } = useDrag<InsertionPoint | undefined>({
  order: () => props.fields,
  nowhere: undefined,
  doesMove: (held, lands) => landing(props.fields, held, lands),
  moves: (held, lands) => emit('move', held, lands),
})

const rows = computed(() => fieldRows(props.fields, dragged.value))

/** What is wrong with one field, and nothing where nothing is. */
const wrongWith = (field: string): readonly string[] => props.wrong?.get(field) ?? []

const add = (): void => {
  emit('add', getFreeName(props.fields, props.words.fieldStem))
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
      <Field
        v-for="row in rows"
        :key="row.field"
        :row="row"
        :naming="naming"
        :objects-id="objectsId(row.field)"
        :wrong="wrongWith(row.field)"
        :before="row.field === at"
        :words="words"
        @over="over"
        @drop="drop"
        @lift="(press) => lift(row.field, press)"
        @release="release"
        @step="(direction, press) => step(row.field, direction, press)"
        @remove="emit('remove', row.field)"
      />
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
</style>
