<script setup lang="ts">
/**
 * The dot after the title of the tab: the state of the run in its colour, named for a screen reader,
 * and the details of the run in a tooltip on hover and on focus.
 */
import { ref, useId } from 'vue'
import type { StatusView } from '../lib/view'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  status: StatusView
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const tooltipId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onShow() {
  isOpen.value = true
}

function onHide() {
  isOpen.value = false
}
</script>

<template>
  <span class="saved-search__status" @pointerenter="onShow" @pointerleave="onHide">
    <span
      role="img"
      tabindex="0"
      class="saved-search__dot"
      :class="`saved-search__dot--${status.state}`"
      :aria-label="status.label"
      :aria-describedby="isOpen ? tooltipId : undefined"
      @focus="onShow"
      @blur="onHide"
      @keydown.escape="onHide"
    />
    <span v-if="isOpen" :id="tooltipId" role="tooltip" class="saved-search__tooltip">
      <span v-for="line in status.details" :key="line" class="saved-search__tooltip-line">
        {{ line }}
      </span>
    </span>
  </span>
</template>
