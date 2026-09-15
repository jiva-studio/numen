<script setup lang="ts">
/**
 * The heading one section of a deck stands under: a rule across the grid with
 * the section's name typed on it, and the way to be rid of it at its end.
 *
 * A section is a name and nothing else. Two of them may carry one name, so what
 * is typed here is measured against nothing, and only a name with nothing in it
 * is refused.
 */
import { computed, useId } from 'vue'
import { ErrorMessage } from '../../error-message'
import { SectionName } from './section-name'
import { Divider } from '../../divider'
import { useNaming } from '../../../model/naming'
import { DECK_WORDS, type DeckWords } from '../../../lib/deck'
import type { PlacedSection } from '../../../lib/grid'
import { checkHeadingName, type HeadingObjection } from '../../../lib/order'

const props = withDefaults(
  defineProps<{
    /** The section, and where it stands among them. */
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

/** What this heading's objection is named by, which is this heading's alone. */
const uid = useId()

const objectionsId = `${uid}-objections`

/** A name typed over the one this section carries, until it is committed. */
const naming = useNaming<HeadingObjection>({
  getName: () => props.section.name,
  getTakenNames: () => [],
  checkName: checkHeadingName,
  rename: (_over, name) => emit('rename', name),
})

/** What is in the box: the name it carries, or what is being typed over it. */
const text = computed(() => naming.getText(props.section.id))

/** Why what is in the box cannot be used, and nothing while it can. */
const objections = computed(() => naming.getObjection(props.section.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => (objections.value === null ? null : props.words.sectionObjection))

/**
 * What the section is announced by. A section's name is a person's own text and
 * may be nothing at all, so what it is called here is the place it stands in.
 */
const stem = computed(() => `${props.words.sectionStem} ${props.section.at}`)
</script>

<template>
  <div class="section-heading">
    <Divider>
      <!-- What a person reaches for is the name and the way to be rid of it,
           and nothing of the line either side. -->
      <SectionName
        :naming="naming"
        :over="section.id"
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

/* What is wrong stands under the rule it is wrong about. */
.section-heading__objections {
  margin: 0;
  padding-inline: 0.375rem;
  overflow-wrap: anywhere;
}
</style>
