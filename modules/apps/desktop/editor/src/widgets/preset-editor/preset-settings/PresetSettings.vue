<script setup lang="ts">
/**
 * The settings a preset's goal schedules by, a row to each. A row is drawn in
 * the control its field takes, held to the ends the application answered with,
 * and written when the person is done with it.
 *
 * A row carries `data-preset-row`, the field it is about. `data-preset` names
 * the parts of one: `name`, `detail`, `day`, `choice` and `percent`.
 */
import { computed } from 'vue'
import type { PresetTabState } from '../composables/usePresetTab'
import { fieldsUnder } from '../curve'
import { WORDS as words } from '../words'
import PresetSettingControl from './PresetSettingControl.vue'

// --- Props & Emits ---
const props = defineProps<{ state: PresetTabState }>()

// --- State ---
const { settings } = props.state

/** The rows the chosen goal schedules by, which are the ones drawn. */
const fields = computed(() => fieldsUnder(settings.value.goal, settings.value.learned))

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <section class="preset-settings" :aria-label="words.settings">
    <div
      v-for="field in fields"
      :key="field"
      class="preset-settings__row"
      :data-preset-row="field"
    >
      <span class="preset-settings__said">
        <span class="preset-settings__name" :id="`preset-${field}`" data-preset="name">{{
          words.fieldName(field)
        }}</span>
        <span class="preset-settings__detail" data-preset="detail">{{
          words.fieldDetail(field)
        }}</span>
      </span>

      <PresetSettingControl :field="field" :state="props.state" />
    </div>
  </section>
</template>

<style scoped>
/* Every row is one grid, so a control begins on the line the control above it
   begins on and every line of prose wraps at the one measure. The control
   column is as wide as the widest control the rows hold. */
.preset-settings {
  /* One row: the box a number is typed into, the list a rule is taken from,
     the air around the row, and the space between what it is called and what
     it means. */
  --preset-settings-value: 6rem;
  --preset-settings-choice: 10rem;
  --preset-settings-air: 0.5rem;
  --preset-settings-said-gap: 0.125rem;
  /* The track a share is moved along, and the room the figure beside it takes. */
  --preset-settings-track: 9rem;
  --preset-settings-percent: 2.25rem;
  display: grid;
  grid-template-columns: 1fr max-content;
}

/* The name and what it means on the left, the control on the right. The row
   takes the settings' own two columns, so nothing is measured row by row. */
.preset-settings__row {
  display: grid;
  grid-column: 1 / -1;
  grid-template-columns: subgrid;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--preset-settings-air);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

/* What the row is called, and under it what it means. */
.preset-settings__said {
  display: flex;
  flex-direction: column;
  gap: var(--preset-settings-said-gap);
  min-inline-size: 0;
}

/* The name of a row and what it means are one size, and the name carries the
   weight and the colour that tell them apart. */
.preset-settings__name {
  color: var(--numen-ink);
  font-weight: 500;
}

.preset-settings__detail {
  color: var(--numen-hushed);
}
</style>
