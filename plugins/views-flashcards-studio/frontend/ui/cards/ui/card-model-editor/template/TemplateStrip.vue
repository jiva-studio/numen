<script setup lang="ts">
/**
 * The strip a template is led by: its name, the row of fields that may be written
 * into it, and what is wrong with it.
 */
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { NameBox } from '../../name-box'
import { useNameEntry } from '../../../model/nameEntry'
import { Button } from '@numen/ui'
import { checkHeadingName, type HeadingObjection } from '../../../lib/order'
import { CARD_MODEL_WORDS, type TemplateRow, type CardModelWords } from '../../../lib/card-model'

const props = withDefaults(
  defineProps<{
    /** The template, laid out against the fields the card model declares. */
    template: TemplateRow
    /** What the caller found wrong with this template, said beside its name. */
    wrong?: readonly string[]
    /** The words it is drawn with. */
    words?: CardModelWords
  }>(),
  { wrong: () => [], words: () => CARD_MODEL_WORDS },
)

const emit = defineEmits<{
  (event: 'rename', name: string): void
  /** A field was pressed, to be written into the template. */
  (event: 'put', field: string): void
}>()

/** What this template's objection is named by, which is this template's alone. */
const uid = useId()

const objectionsId = `${uid}-objections`

/** A name typed over the one this template carries, until it is committed. */
const naming = useNameEntry<HeadingObjection>({
  getName: () => props.template.name,
  getTakenNames: () => props.template.taken,
  checkName: checkHeadingName,
  rename: (_targetId, name) => emit('rename', name),
})

/** Why what is in the name box cannot be used, and nothing while it can. */
const objections = computed(() => naming.getObjection(props.template.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => {
  const objection = objections.value
  return objection === null ? null : props.words.templateObjection(objection)
})
</script>

<template>
  <div class="template__said">
    <div class="template__head flex items-center">
      <NameBox
        class="template__title rounded-node"
        :naming="naming"
        :target-id="template.id"
        :stem="`${words.templateStem} ${template.at}`"
        :described-by="says ? objectionsId : null"
      />

      <!-- The fields are small quiet chips, as small quiet actions are
           drawn everywhere else here. What they are for is said to a
           reader by the group, and to everyone else by their look. -->
      <div class="template__slots flex" role="group" :aria-label="`${words.insert}: ${template.name}`">
        <Button
          v-for="field in template.fields"
          :key="field"
          variant="ghost"
          size="small"
          class="template__slot rounded-pill bg-bubble text-small h-5 px-2"
          draggable="false"
          :data-insert="field"
          :aria-label="`${words.insert}: ${field}`"
          @click="emit('put', field)"
        >
          {{ field }}
        </Button>
      </div>
    </div>

    <!-- What is wrong with the template stands at the end of the strip, over
         the window under it. -->
    <div v-if="says || wrong.length" class="template__amiss">
      <ErrorMessage
        v-if="says"
        :id="objectionsId"
        class="template__objections"
        role="alert"
        :said="says"
      />

      <ErrorMessage
        v-if="wrong.length"
        class="template__objections"
        data-wrong
        :said="wrong"
        :label="words.wrong"
      />
    </div>
  </div>
</template>

<style scoped>
/* The strip is the template's own, so its name leads it and the fields follow. */
.template__said {
  inline-size: 100%;
}

.template__head {
  inline-size: 100%;
  gap: var(--numen-inset);
}

/* The name is the heading of the template: the largest thing in the strip, and
   never squeezed by however many fields stand beside it. */
.template__title {
  flex: 0 1 12rem;
  min-inline-size: 5rem;
  font-weight: 500;
}

/* The fields are a group under the heading, not its equal. A long row scrolls
   inside the strip. */
.template__slots {
  flex: 1 1 auto;
  min-inline-size: 0;
  gap: 0.25rem;
  overflow-x: auto;
  scrollbar-width: none;
}

.template__slots::-webkit-scrollbar {
  display: none;
}

.template__slot:hover {
  background: color-mix(in oklab, var(--numen-bubble-bg), var(--numen-ink) 10%);
}

/* The ring is drawn inside the chip, so the row it scrolls in cannot clip it. */
.template__slot:focus-visible {
  box-shadow: none;
  outline: none;
}

/* What is wrong with the template is placed at the bottom right, stacked over any pane errors. */
.template__amiss {
  position: absolute;
  z-index: 5;
  inset-block-end: var(--box-air, 0.5rem);
  inset-inline-end: var(--box-pad-inline, 0.5rem);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.125rem;
  max-inline-size: calc(100% - 2 * var(--box-pad-inline, 0.5rem));
  pointer-events: none;
}

:global(.template:has(.template__pane .template__amiss)) .template__amiss {
  inset-block-end: calc(var(--box-air, 0.5rem) + 1.875rem);
}

/* What is wrong is read over whatever it covers, so it carries a ground. */
.template__amiss > .template__objections {
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius);
  background: var(--numen-alarm-bg);
}
</style>
