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
import { RemoveButton } from '../../remove-button'
import { NameBox } from '../../name-box'
import { Divider } from '../../divider'
import { useNaming } from '../../naming'
import { DECK_WORDS, type DeckWords, type PlacedSection } from '../../deck'
import { heading, type Refusal } from '../../order'

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

const objectsId = `${uid}-objects`

/** A name typed over the one this section carries, until it is committed. */
const naming = useNaming<Refusal>({
  carries: () => props.section.name,
  taken: () => [],
  amiss: heading,
  renamed: (_over, name) => emit('rename', name),
})

/** What is in the box: the name it carries, or what is being typed over it. */
const text = computed(() => naming.text(props.section.id))

/** Why what is in the box cannot be used, and nothing while it can. */
const objects = computed(() => naming.objection(props.section.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => (objects.value === null ? null : props.words.sectionObjection))

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
      <span class="section-heading__held">
        <span class="section-heading__name" :data-typed="text || stem">
          <!-- The box is as wide as the cell behind it comes to, and the cell
               is set to the text. -->
          <NameBox
            class="section-heading__title min-w-0 rounded-node"
            :naming="naming"
            :over="section.id"
            :stem="stem"
            :described-by="says ? objectsId : null"
          />
        </span>

        <span class="section-heading__actions">
          <RemoveButton :label="`${words.remove}: ${stem}`" @press="emit('remove')" />
        </span>
      </span>
    </Divider>

    <ErrorMessage
      v-if="says"
      :id="objectsId"
      class="section-heading__objects"
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

/* What a person reaches for is the name and what stands at its end, and it is
   as wide as the two of them come to. */
.section-heading__held {
  display: flex;
  align-items: center;
  min-inline-size: 0;
}

/* The box is as wide as what is typed in it: the same text is set behind the
   input, unseen, and the box takes the width it comes to. Past twenty
   characters' room the text scrolls inside. */
.section-heading__name {
  display: inline-grid;
  flex: 0 1 auto;
  min-inline-size: 0;
  max-inline-size: 20rem;
}

.section-heading__name::after,
.section-heading__title {
  grid-area: 1 / 1;
  font: inherit;
  font-weight: 500;
}

/* The cell behind the box is the box's own size, so the two hold the same air. */
.section-heading__name::after {
  content: attr(data-typed);
  padding: var(--card-row-pad-block, 0.125rem) var(--card-row-pad-inline, 0.375rem);
  visibility: hidden;
  white-space: pre;
}

/* The name is the heading of everything below it, and stands in the middle of
   the rule it is typed on. */
.section-heading__title {
  text-align: center;
}

/* What the section is pressed to be rid of is not drawn until its name is
   reached for, by the pointer or by the keyboard. Until then it takes no room
   at all, and the line runs unbroken up to the name. */
.section-heading__actions {
  display: flex;
  flex: none;
  align-items: center;
  inline-size: 0;
  overflow: hidden;
  opacity: 0;
  will-change: opacity;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.section-heading__held:hover .section-heading__actions,
.section-heading__held:focus-within .section-heading__actions {
  inline-size: auto;
  padding-inline-start: var(--numen-inset);
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .section-heading__actions {
    transition: none;
  }
}

/* What is wrong stands under the rule it is wrong about. */
.section-heading__objects {
  margin: 0;
  padding-inline: 0.375rem;
  overflow-wrap: anywhere;
}
</style>
