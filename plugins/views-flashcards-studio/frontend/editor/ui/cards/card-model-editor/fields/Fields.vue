<script setup lang="ts">
import { computed, useId } from 'vue'
import { Icon } from '../../icon'
import { Divider } from '../../divider'
import Field from './Field.vue'
import { useDrag } from '../../../../model/cards/drag'
import { useNameEntry } from '../../../../model/cards/nameEntry'
import { Button } from '@numen/ui'
import {
  canDropField,
  getFreeName,
  checkFieldName,
  type Problems,
  type InsertionPoint,
  type Objection,
} from '../../../../../lib/cards'
import { fieldRows, CARD_MODEL_WORDS, type CardModelWords } from '../../../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    fields: readonly string[]
    wrong?: Problems | null
    words?: CardModelWords
  }>(),
  { wrong: null, words: () => CARD_MODEL_WORDS },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'add', name: string): void
  (event: 'rename', field: string, name: string): void
  (event: 'remove', field: string): void
  (event: 'move', field: string, target: InsertionPoint): void
}>()

/* ---------------------------------- State --------------------------------- */
const uid = useId()

const naming = useNameEntry<Objection>({
  getName: (field) => field,
  getTakenNames: (field) => props.fields.filter((each) => each !== field),
  checkName: checkFieldName,
  rename: (field, name) => emit('rename', field, name),
})

const {
  dragged,
  at: dropTarget,
  startDrag,
  hover,
  release,
  drop,
  step,
} = useDrag<InsertionPoint | undefined>({
  order: () => props.fields,
  nowhere: undefined,
  isMoved: (dragged, lands) => canDropField(props.fields, dragged, lands),
  move: (dragged, lands) => emit('move', dragged, lands),
})

const rows = computed(() => fieldRows(props.fields, dragged.value))

/* -------------------------------- Handlers -------------------------------- */
const onAdd = (): void => {
  emit('add', getFreeName(props.fields, props.words.fieldStem))
}

/* --------------------------------- Helpers -------------------------------- */
const objectionsId = (targetId: string): string =>
  `${uid}-${encodeURIComponent(targetId)}-objections`

const wrongWith = (field: string): readonly string[] => props.wrong?.get(field) ?? []
</script>

<template>
  <section
    class="card-model__part"
    :aria-label="words.fields"
    @dragover="hover(null, $event)"
    @drop="drop"
  >
    <h2 class="card-model__heading caps-numen text-small text-hushed m-0">{{ words.fields }}</h2>

    <ul v-if="rows.length" class="card-model__fields">
      <Field
        v-for="row in rows"
        :key="row.field"
        :row="row"
        :naming="naming"
        :objections-id="objectionsId(row.field)"
        :wrong="wrongWith(row.field)"
        :is-drop-above="row.field === dropTarget"
        :words="words"
        @drag-over="hover"
        @drop="drop"
        @drag-start="(press) => startDrag(row.field, press)"
        @release="release"
        @step="(direction, press) => step(row.field, direction, press)"
        @remove="emit('remove', row.field)"
      />
    </ul>

    <p v-else class="card-model__silence caps-numen text-small text-hushed m-0">
      {{ words.noFields }}
    </p>

    <Divider>
      <Button variant="ghost" size="small" @click="onAdd">
        <Icon name="plus" />
        {{ words.addField }}
      </Button>
    </Divider>
  </section>
</template>

<style scoped>
.card-model__fields {
  display: flex;
  flex-direction: column;
  gap: var(--row-gap);
  inline-size: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
