<script setup lang="ts">
/**
 * A card model editor: the fields it names at the top and the templates that
 * display them below, the first field naming the cards this card model cuts.
 *
 * The caller owns the card model and its problems: a template's problems are
 * shown next to that template's name, and a field's problems under that field's
 * row.
 */
import { computed } from 'vue'
import { CardTemplate } from './template'
import { Fields } from './fields'
import { AddTemplate } from './add-template'
import { useDrag } from '../../model/drag'
import { getDeclaredFields, getFreeName, type Half, type InsertionPoint } from '../../lib/order'
import {
  templateRows,
  NO_CARD_MODEL_PROBLEMS,
  CARD_MODEL_WORDS,
  type CardModelTemplate,
  type CardModelWords,
  type CardModelProblems,
} from '../../lib/card-model'
import { sampleValues } from '../../lib/fill'

/**
 * The problems the caller found with the card model. A template's problems are
 * shown next to that template's name and a field's problems under that field's
 * row, so every problem appears next to the element it refers to.
 */
/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** The fields, in the order the user supplies them. The first names the card. */
    fields: readonly string[]
    /** The templates, in the order they are drawn. */
    templates: readonly CardModelTemplate[]
    /** What the editor is announced as. */
    name?: string
    /** What the caller found wrong with the fields and the templates it handed in. */
    wrong?: CardModelProblems
    /** The words it is drawn with. */
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
  /** A field let go somewhere in the order: before another, or at the end. */
  (event: 'move-field', field: string, target: InsertionPoint): void
  (event: 'add-template', name: string): void
  (event: 'rename-template', id: string, name: string): void
  (event: 'remove-template', id: string): void
  /**
   * A template let go somewhere in the order: before another, or at the end. The
   * order of the templates is the order a card's repetitions are taken from it, so
   * nothing among them is fixed.
   */
  (event: 'move-template', id: string, target: InsertionPoint): void
  /** One half of one template, with its current text. */
  (event: 'write', id: string, half: Half, text: string): void
}>()

/* ---------------------------------- State --------------------------------- */
/** The fields a card requires; a name declared twice counts as one field. */
const declaredFields = computed(() => getDeclaredFields(props.fields))

/**
 * The template being dragged, and the position where releasing it would place it.
 * The order of the templates is the order in which a card's repetitions are
 * taken, so no template has a fixed position and any template can be dropped at
 * any position other than its current one.
 */
const {
  dragged: template,
  at: templateAt,
  lift: liftTemplate,
  hover: hoverTemplate,
  release: releaseTemplate,
  drop: dropTemplate,
  step: stepTemplate,
} = useDrag<InsertionPoint>({
  order: () => props.templates.map((each) => each.id),
  nowhere: null,
  isMoved: (held, lands) => lands !== held,
  move: (held, lands) => emit('move-template', held, lands),
})

const sample = computed(() => sampleValues(declaredFields.value))

/** The templates as they are drawn. */
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
/** What is wrong with one template, and nothing where nothing is. */
const wrongWithTemplate = (id: string): readonly string[] => props.wrong.at.get(id) ?? []
</script>

<template>
  <div
    class="card model numen bg-surface text-ink font-sans text-base"
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
        @lift="liftTemplate(one.id, $event)"
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

.cardModel {
  /* The space between one part and the next, and between the rows of a part. */
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
