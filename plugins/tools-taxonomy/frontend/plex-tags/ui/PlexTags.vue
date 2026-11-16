<script setup lang="ts">
/** The tags of a note under its node in the plex: the first four as chips with their names and the count of the rest. */
import { computed } from 'vue'
import { getTagColor } from '../../lib/color'
import { getRestText } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ tags: readonly string[] }>()

/* --------------------------------- State ---------------------------------- */
const CHIP_LIMIT = 4

const chips = computed(() =>
  props.tags.slice(0, CHIP_LIMIT).map((tag, index) => ({
    tag,
    style: { '--tag': `var(--numen-tag-${getTagColor(tag)})`, '--i': index },
  })),
)
const rest = computed(() => props.tags.length - CHIP_LIMIT)
</script>

<template>
  <TransitionGroup
    v-if="tags.length > 0"
    tag="span"
    name="plex-tags"
    appear
    class="plex-tags"
    data-plex-tags
  >
    <span
      v-for="chip in chips"
      :key="chip.tag"
      class="plex-tags__chip"
      data-plex-tag
      :style="chip.style"
    >
      {{ chip.tag }}
    </span>
    <span
      v-if="rest > 0"
      key="more"
      class="plex-tags__chip plex-tags__chip--more"
      data-plex-tag-more
      :style="{ '--i': CHIP_LIMIT }"
    >
      {{ getRestText(rest) }}
    </span>
  </TransitionGroup>
</template>

<style scoped>
.plex-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--numen-node-gap);
  max-inline-size: 100%;
  margin-block-start: var(--numen-node-gap);
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
  overflow-wrap: anywhere;
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

.plex-tags-enter-active {
  transition:
    opacity var(--numen-motion) var(--numen-easing),
    transform var(--numen-motion) var(--numen-easing);
  transition-delay: calc(var(--i) * 30ms);
}

.plex-tags-leave-active {
  position: absolute;
  transition: opacity var(--numen-motion) var(--numen-easing);
}

.plex-tags-move {
  transition: transform var(--numen-motion) var(--numen-easing);
}

.plex-tags-enter-from {
  opacity: 0.2;
  transform: translateY(-0.25rem) scale(0.9);
}

.plex-tags-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .plex-tags-enter-active,
  .plex-tags-leave-active,
  .plex-tags-move {
    transition: none;
  }
}
</style>
