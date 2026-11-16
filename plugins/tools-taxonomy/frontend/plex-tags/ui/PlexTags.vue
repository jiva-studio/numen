<script setup lang="ts">
/** The tags of a note under its node in the plex, each as a chip with its name. */
import { computed } from 'vue'
import { getTagColor } from '../../lib/color'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ tags: readonly string[] }>()

/* --------------------------------- State ---------------------------------- */
const chips = computed(() =>
  props.tags.map((tag) => ({
    tag,
    style: { '--tag': `var(--numen-tag-${getTagColor(tag)})` },
  })),
)
</script>

<template>
  <span v-if="tags.length > 0" class="plex-tags" data-plex-tags>
    <span
      v-for="(chip, index) in chips"
      :key="index"
      class="plex-tags__chip"
      data-plex-tag
      :style="chip.style"
    >
      {{ chip.tag }}
    </span>
  </span>
</template>

<style scoped>
.plex-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
  font-size: var(--numen-text-1);
  line-height: 1.2;
}

.plex-tags__chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.0625rem 0.375rem;
  border-radius: var(--numen-radius-pill);
  background: color-mix(in srgb, var(--tag) 18%, transparent);
  color: var(--numen-ink);
}

.plex-tags__chip::before {
  content: '';
  inline-size: 0.5rem;
  block-size: 0.5rem;
  border-radius: 50%;
  background: var(--tag);
}
</style>
