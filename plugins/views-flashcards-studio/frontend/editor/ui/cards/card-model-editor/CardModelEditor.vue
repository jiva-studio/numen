<script setup lang="ts">
import { computed } from 'vue'
import { CardTemplate } from './template'
import { Fields } from './fields'
import { AddTemplate } from './add-template'
import { useDrag } from '../../../model/cards/drag'
import {
  getDeclaredFields,
  getFreeName,
  type Half,
  type InsertionPoint,
} from '../../../../lib/cards'
import {
  templateRows,
  NO_CARD_MODEL_PROBLEMS,
  CARD_MODEL_WORDS,
  type CardModelTemplate,
  type CardModelWords,
  type CardModelProblems,
} from '../../../../lib/cards'
import { sampleValues } from '../../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    fields: readonly string[]
    templates: readonly CardModelTemplate[]
    name?: string
    wrong?: CardModelProblems
    words?: CardModelWords
  }>(),
  {
    name: 'Card model',
    wrong: () => NO_CARD_MODEL_PROBLEMS,
    words: () => CARD_MODEL_WORDS,
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'add-field', name: string): void
  (event: 'rename-field', field: string, name: string): void
  (event: 'remove-field', field: string): void
  (event: 'move-field', field: string, target: InsertionPoint): void
  (event: 'add-template', name: string): void
  (event: 'rename-template', id: string, name: string): void
  (event: 'remove-template', id: string): void
  (event: 'move-template', id: string, target: InsertionPoint): void
  (event: 'write', id: string, half: Half, text: string): void
}>()

/* ---------------------------------- State --------------------------------- */
const declaredFields = computed(() => getDeclaredFields(props.fields))

const {
  dragged: template,
  at: templateAt,
  startDrag: startTemplateDrag,
  hover: hoverTemplate,
  release: releaseTemplate,
  drop: dropTemplate,
  step: stepTemplate,
} = useDrag<InsertionPoint>({
  order: () => props.templates.map((each) => each.id),
  nowhere: null,
  isMoved: (dragged, lands) => lands !== dragged,
  move: (dragged, lands) => emit('move-template', dragged, lands),
})

const sample = computed(() => sampleValues(declaredFields.value))

const renderedTemplates = computed(() =>
  templateRows(props.templates, declaredFields.value, sample.value),
)

/* -------------------------------- Handlers -------------------------------- */
const onAddTemplate = (): void => {
  emit(
    'add-template',
    getFreeName(
      props.templates.map((each) => each.name),
      props.words.templateStem,
    ),
  )
}

/* --------------------------------- Helpers -------------------------------- */
const wrongWithTemplate = (id: string): readonly string[] => props.wrong.at.get(id) ?? []
</script>

<template>
  <div
    class="card card-model numen bg-surface text-ink font-sans text-base"
    role="group"
    :aria-label="name"
  >
    <Fields
      :fields="declaredFields"
      :wrong="wrong.fields"
      :words="words"
      @add="(name: string) => emit('add-field', name)"
      @rename="(field: string, name: string) => emit('rename-field', field, name)"
      @remove="(field: string) => emit('remove-field', field)"
      @move="(field: string, at: InsertionPoint) => emit('move-field', field, at)"
    />

    <section
      class="card-model__part"
      :aria-label="words.templates"
      @dragover="hoverTemplate(null, $event)"
      @drop="dropTemplate"
    >
      <h2 class="card-model__heading caps-numen text-small text-hushed m-0">
        {{ words.templates }}
      </h2>

      <p
        v-if="!renderedTemplates.length"
        class="card-model__silence caps-numen text-small text-hushed m-0"
      >
        {{ words.noTemplates }}
      </p>

      <CardTemplate
        v-for="one in renderedTemplates"
        :key="one.id"
        class="card-model__template caret-above"
        :template="one"
        :wrong="wrongWithTemplate(one.id)"
        :words="words"
        :data-dragged="one.id === template || undefined"
        :data-before="one.id === templateAt || undefined"
        @dragover.stop="hoverTemplate(one.id, $event)"
        @drop.stop="dropTemplate"
        @rename="(name: string) => emit('rename-template', one.id, name)"
        @remove="emit('remove-template', one.id)"
        @drag-start="startTemplateDrag(one.id, $event)"
        @release="releaseTemplate"
        @step="(direction, press) => stepTemplate(one.id, direction, press)"
        @write="(half, text) => emit('write', one.id, half, text)"
      />

      <AddTemplate :label="words.addTemplate" @press="onAddTemplate" />
    </section>
  </div>
</template>

<style scoped>
@import '../caret.css';

.card-model {
  --part-gap: 1.5rem;
  --row-gap: 0.5rem;
  --caret-out: calc(-1 * var(--row-gap) / 2);

  display: flex;
  flex-direction: column;
  gap: var(--part-gap);
  padding: var(--numen-gutter);
  overflow: auto;
}

.card-model__part {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--row-gap);
}

.card-model__template[data-dragged] {
  opacity: 0.5;
}
</style>
