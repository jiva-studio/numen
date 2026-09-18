<script setup lang="ts">
/**
 * What a person reaches for in a section's heading: the name, typed into where
 * it stands, and at its end the way to be rid of the section.
 */
import { NameBox } from '../../../name-box'
import { RemoveButton } from '../../../remove-button'
import type { NameEntryState } from '../../../../model/naming'
import type { Objection } from '../../../../lib/order'

defineProps<{
  /** The naming this box types into. */
  naming: NameEntryState<Objection>
  /** What is being typed over, as the naming addresses it. */
  over: string
  /** What the section is announced by while it carries no name. */
  stem: string
  /** What is in the box, which the cell behind it is set to. */
  text: string
  /** What says why the name cannot be used, and nothing while it can. */
  describedBy: string | null
  /** What the button to be rid of the section is announced as. */
  removeLabel: string
}>()

const emit = defineEmits<{
  (event: 'remove'): void
}>()
</script>

<template>
  <span class="section-heading__held">
    <span class="section-heading__name" :data-typed="text || stem">
      <!-- The box is as wide as the cell behind it comes to, and the cell
           is set to the text. -->
      <NameBox
        class="section-heading__title rounded-node min-w-0"
        :naming="naming"
        :over="over"
        :stem="stem"
        :described-by="describedBy"
      />
    </span>

    <span class="section-heading__actions">
      <RemoveButton :label="removeLabel" @press="emit('remove')" />
    </span>
  </span>
</template>

<style scoped>
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
</style>
