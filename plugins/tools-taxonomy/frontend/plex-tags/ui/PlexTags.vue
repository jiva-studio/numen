<script setup lang="ts">
/** The tags of a note under its node in the plex: the first four as chips with their names and the count of the rest. */
import { computed, ref, watch } from 'vue'
import { measureRect, Tooltip, type Box } from '@numen/ui'
import { getTagColor } from '../../lib/color'
import { getRestText } from '../../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ tags: readonly string[] }>()

/* --------------------------------- State ---------------------------------- */
const CHIP_LIMIT = 4

const hoveredAt = ref<Box | null>(null)

const chips = computed(() =>
  props.tags.slice(0, CHIP_LIMIT).map((tag) => ({
    tag,
    style: { '--tag': `var(--numen-tag-${getTagColor(tag)})` },
  })),
)
const rest = computed(() => props.tags.length - CHIP_LIMIT)
const hiddenTags = computed(() => props.tags.slice(CHIP_LIMIT))
const hoveredText = computed(() => hiddenTags.value.join(', '))

watch(
  () => props.tags,
  (tags) => {
    if (tags.length <= CHIP_LIMIT) hoveredAt.value = null
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onEnterMore(event: Event) {
  const rect = measureRect(event.currentTarget as HTMLElement)
  hoveredAt.value = rect ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null
}

function onLeaveMore() {
  hoveredAt.value = null
}
</script>

<template>
  <span v-if="tags.length > 0" class="plex-tags" data-plex-tags>
    <span
      v-for="chip in chips"
      :key="chip.tag"
      class="plex-tags__chip"
      data-plex-tag
      :style="chip.style"
    >
      <span class="plex-tags__name">{{ chip.tag }}</span>
    </span>
    <span
      v-if="rest > 0"
      class="plex-tags__chip plex-tags__chip--more"
      data-plex-tag-more
      tabindex="0"
      @mouseenter="onEnterMore"
      @mouseleave="onLeaveMore"
      @focus="onEnterMore"
      @blur="onLeaveMore"
    >
      {{ getRestText(rest) }}
    </span>
    <Teleport to="body">
      <Tooltip v-if="hoveredAt" :at="hoveredAt">{{ hoveredText }}</Tooltip>
    </Teleport>
  </span>
</template>

<style scoped>
.plex-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--numen-node-gap);
  margin-block-start: var(--numen-node-gap);
  max-inline-size: 100%;
  font-size: var(--numen-text-1);
  line-height: 1.2;
}

.plex-tags__chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  max-inline-size: 100%;
  padding: 0.0625rem 0.375rem;
  border-radius: var(--numen-radius-pill);
  background: color-mix(in srgb, var(--tag) 18%, transparent);
  color: var(--numen-ink);
  white-space: nowrap;
}

.plex-tags__name {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.plex-tags__chip::before {
  content: '';
  flex: none;
  inline-size: 0.5rem;
  block-size: 0.5rem;
  border-radius: 50%;
  background: var(--tag);
}

.plex-tags__chip--more {
  background: color-mix(in srgb, currentColor 12%, transparent);
}

.plex-tags__chip--more::before {
  content: none;
}
</style>
