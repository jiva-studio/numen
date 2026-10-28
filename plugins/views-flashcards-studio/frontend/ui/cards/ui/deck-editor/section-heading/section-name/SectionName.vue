<script setup lang="ts">
import { NameBox } from '../../../name-box'
import { RemoveButton } from '../../../remove-button'
import type { NameEntryState } from '../../../../model/nameEntry'
import type { Objection } from '../../../../lib/order'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  /** Name entry state managing this section heading input. */
  naming: NameEntryState<Objection>
  /** Unique target identifier for the section. */
  targetId: string
  /** Placeholder and accessible name when title is empty. */
  stem: string
  /** Current section name text. */
  text: string
  /** Validation description element ID. */
  describedBy: string | null
  /** Accessible label for the remove button. */
  removeLabel: string
}>()

const emit = defineEmits<{
  (event: 'remove'): void
}>()
</script>

<template>
  <span class="section-heading__held">
    <span class="section-heading__name" :data-typed="text || stem">
      <NameBox
        class="section-heading__title rounded-node min-w-0"
        :naming="naming"
        :target-id="targetId"
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
.section-heading__held {
  display: flex;
  align-items: center;
  min-inline-size: 0;
}

/* Sizing container matches width to mirror content up to max-inline-size. */
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

/* Hidden mirror pseudo-element matching input text for auto-sizing. */
.section-heading__name::after {
  content: attr(data-typed);
  padding: var(--card-row-pad-block) var(--card-row-pad-inline);
  visibility: hidden;
  white-space: pre;
}

.section-heading__title {
  text-align: center;
}

/* Reveal action buttons on hover or focus within. */
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
