<script setup lang="ts">
/**
 * The dot after the title of the tab: the state of the run in its colour, named for a screen reader,
 * and a card of the run in a tooltip on hover and on focus: the state as its title, the facts of the
 * run as label and value rows with the times as they read from now, and the error and the warnings
 * under a rule.
 */
import { ref } from 'vue'
import { Tooltip, useTooltip } from '@numen/ui'
import { WORDS as words } from '../../words'
import { formatRelativeTime } from '../lib/time'
import type { StatusFact, StatusView } from '../lib/view'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  status: StatusView
  getNow: () => number
}>()

/* --------------------------------- State ---------------------------------- */
const tip = useTooltip()
const openedAt = ref(0)

/* -------------------------------- Handlers -------------------------------- */
function onShow(event: Event) {
  openedAt.value = props.getNow()
  tip.open(event)
}

function onHide() {
  tip.close()
}

/* --------------------------------- Helpers -------------------------------- */
function getValue(fact: StatusFact): string {
  if (fact.at === '') return fact.value
  return formatRelativeTime(fact.at, openedAt.value, props.status.locale)
}
</script>

<template>
  <div class="saved-search__status">
    <span
      role="img"
      tabindex="0"
      class="saved-search__dot"
      :class="`saved-search__dot--${status.state}`"
      :aria-label="status.label"
      :aria-describedby="tip.describedBy.value"
      @mouseenter="onShow"
      @mouseleave="onHide"
      @focus="onShow"
      @blur="onHide"
      @keydown.escape="onHide"
    />
    <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value" class="saved-search__tooltip">
      <p class="saved-search__tooltip-title">
        {{ status.label }}
      </p>
      <dl v-if="status.facts.length > 0" class="saved-search__tooltip-facts">
        <template v-for="fact in status.facts" :key="fact.label">
          <dt class="saved-search__tooltip-label">{{ fact.label }}</dt>
          <dd class="saved-search__tooltip-value">
            {{ getValue(fact) }}
          </dd>
        </template>
      </dl>
      <section
        v-if="status.errors.length > 0"
        class="saved-search__tooltip-section saved-search__tooltip-section--error"
      >
        <h3 class="saved-search__tooltip-heading">{{ words.error }}</h3>
        <ul v-if="status.errors.length > 1" class="saved-search__tooltip-list">
          <li v-for="error in status.errors" :key="error">{{ error }}</li>
        </ul>
        <p v-else class="saved-search__tooltip-message">{{ status.errors[0] }}</p>
      </section>
      <section
        v-if="status.warnings.length > 0"
        class="saved-search__tooltip-section saved-search__tooltip-section--warning"
      >
        <h3 class="saved-search__tooltip-heading">
          {{ status.warnings.length > 1 ? words.warnings : words.warning }}
        </h3>
        <ul v-if="status.warnings.length > 1" class="saved-search__tooltip-list">
          <li v-for="warning in status.warnings" :key="warning">{{ warning }}</li>
        </ul>
        <p v-else class="saved-search__tooltip-message">{{ status.warnings[0] }}</p>
      </section>
    </Tooltip>
  </div>
</template>
