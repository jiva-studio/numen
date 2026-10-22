<script setup lang="ts">
/**
 * Deck section heading component with editable title and removal action.
 */
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { SectionName } from './section-name'
import { Divider } from '../../divider'
import { useNameEntry } from '../../../model/nameEntry'
import { DECK_WORDS, type DeckWords } from '../../../lib/deck'
import type { PlacedSection } from '../../../lib/grid'
import { checkHeadingName, type HeadingObjection } from '../../../lib/order'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    /** The section, and at its current position among them. */
    section: PlacedSection
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'rename', name: string): void
  (event: 'remove'): void
}>()

/* --------------------------------- State ---------------------------------- */
const uid = useId()
const objectionsId = `${uid}-objections`

const naming = useNameEntry<HeadingObjection>({
  getName: () => props.section.name,
  getTakenNames: () => [],
  checkName: checkHeadingName,
  rename: (_targetId, name) => emit('rename', name),
})

const text = computed(() => naming.getText(props.section.id))
const objections = computed(() => naming.getObjection(props.section.id))
const says = computed(() => (objections.value === null ? null : props.words.sectionObjection))
const stem = computed(() => `${props.words.sectionStem} ${props.section.at}`)
</script>

<template>
  <div class="section-heading">
    <Divider>
      <!-- What a person reaches for is the name and the way to be rid of it,
           and nothing of the line either side. -->
      <SectionName
        :naming="naming"
        :target-id="section.id"
        :stem="stem"
        :text="text"
        :described-by="says ? objectionsId : null"
        :remove-label="`${words.remove}: ${stem}`"
        @remove="emit('remove')"
      />
    </Divider>

    <ErrorMessage
      v-if="says"
      :id="objectionsId"
      class="section-heading__objections"
      role="alert"
      :said="says"
    />
  </div>
</template>

<style scoped>
.section-heading {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  inline-size: 100%;
  min-inline-size: 0;
}

/* What is wrong is placed under the rule it is wrong about. */
.section-heading__objections {
  margin: 0;
  padding-inline: 0.375rem;
  overflow-wrap: anywhere;
}
</style>
