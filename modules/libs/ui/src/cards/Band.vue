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
import Amiss from './Amiss.vue'
import Deed from './Deed.vue'
import Rule from '../rule/Rule.vue'
import { useNaming } from './naming'
import { DECK_WORDS, type Band, type DeckWords } from './deck'
import { heading, type Refusal } from './order'

const props = withDefaults(
  defineProps<{
    /** The section, and where it stands among them. */
    band: Band
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'rename', name: string): void
  (event: 'remove'): void
}>()

/** What this band's objection is named by, which is this band's alone. */
const uid = useId()

const objectsId = `${uid}-objects`

/** A name typed over the one this section carries, until it is committed. */
const naming = useNaming<Refusal>({
  carries: () => props.band.name,
  taken: () => [],
  amiss: heading,
  renamed: (_over, name) => emit('rename', name),
})

/** What is in the box: the name it carries, or what is being typed over it. */
const text = computed(() => naming.text(props.band.id))

/** Why what is in the box cannot be used, and nothing while it can. */
const objects = computed(() => naming.objection(props.band.id))

/** What is said of a name that cannot be used, and nothing while it can. */
const says = computed(() => {
  const why = objects.value
  return why === null ? null : props.words.sectionObjection(why)
})

/** What the section is announced by while it carries no name of its own yet. */
const stem = computed(() => `${props.words.sectionStem} ${props.band.at}`)
</script>

<template>
  <div class="band" :data-band-of="band.id">
    <Rule at="start">
      <input
        class="band__title min-w-0 rounded-node"
        type="text"
        :value="text"
        :placeholder="stem"
        :aria-label="stem"
        :aria-invalid="objects !== null || undefined"
        :aria-describedby="says ? objectsId : undefined"
        @input="naming.typing(band.id, ($event.target as HTMLInputElement).value)"
        @change="naming.commit(band.id)"
        @keydown="naming.onKey($event, band.id)"
      />

      <Deed :label="`${words.remove}: ${band.name}`" @press="emit('remove')" />
    </Rule>

    <Amiss v-if="says" :id="objectsId" class="band__objects" role="alert" :said="says" />
  </div>
</template>

<style scoped>
.band {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  inline-size: 100%;
  min-inline-size: 0;
}

/* The name is typed on the rule and carries neither a line nor a ground of its
   own. It is the heading of everything below it, and reads as one. */
.band__title {
  flex: 0 1 20rem;
  min-inline-size: 5rem;
  padding: 0.125rem 0.375rem;
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  font-weight: 500;
  cursor: auto;
}

.band__title:focus-visible {
  outline: none;
}

/* What is wrong stands under the rule it is wrong about. */
.band__objects {
  margin: 0;
  padding-inline: 0.375rem;
  overflow-wrap: anywhere;
}
</style>
