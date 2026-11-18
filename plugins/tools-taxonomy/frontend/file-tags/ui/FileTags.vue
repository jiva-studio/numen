<script setup lang="ts">
/**
 * The tags of a file as colored dots that overlap like avatars, four at most and the count of the rest,
 * with a tooltip that names every tag.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { measureRect, Tooltip, type Box } from '@numen/ui'
import { getTagColor } from '../../lib/color'
import { getRestText } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ tags: readonly string[] }>()

/* --------------------------------- State ---------------------------------- */
const DOT_LIMIT = 4

const root = useTemplateRef<HTMLElement>('root')
const at = ref<Box | null>(null)

const dots = computed(() =>
  props.tags.slice(0, DOT_LIMIT).map((tag) => ({
    tag,
    style: { '--tag': `var(--numen-tag-${getTagColor(tag)})` },
  })),
)
const rest = computed(() => props.tags.length - DOT_LIMIT)
const label = computed(() => props.tags.join(', '))
const data = computed(() => JSON.stringify(props.tags))

/* -------------------------------- Handlers -------------------------------- */
function onEnter() {
  const rect = measureRect(root.value)
  at.value = rect ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null
}

function onLeave() {
  at.value = null
}
</script>

<template>
  <template v-if="tags.length > 0">
    <span
      ref="root"
      class="file-tags"
      data-tag-dots
      :data-tags="data"
      role="img"
      :aria-label="label"
      @mouseenter="onEnter"
      @mouseleave="onLeave"
    >
      <span
        v-for="dot in dots"
        :key="dot.tag"
        class="file-tags__dot"
        data-tag-dot
        :style="dot.style"
      />
      <span v-if="rest > 0" class="file-tags__rest">{{ getRestText(rest) }}</span>
    </span>
    <Teleport to="body">
      <Tooltip v-if="at" :at="at">{{ label }}</Tooltip>
    </Teleport>
  </template>
</template>

<style scoped>
.file-tags {
  --ring: 0.125rem;
  --dot: 0.625rem;

  display: inline-flex;
  align-items: center;
  font-size: var(--numen-text-1);
  line-height: 1;
}

.file-tags__dot {
  box-sizing: content-box;
  inline-size: var(--dot);
  block-size: var(--dot);
  border-radius: 50%;
  background: var(--tag);
  box-shadow: 0 0 0 var(--ring) var(--tree-row-bg);
}

.file-tags__dot + .file-tags__dot {
  margin-inline-start: calc(var(--dot) / -2);
}

.file-tags__rest {
  margin-inline-start: calc(var(--ring) * 2);
}
</style>
