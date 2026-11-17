<script setup lang="ts">
/**
 * The tags of a file as colored dots that overlap like avatars, four at most and the count of the rest,
 * with a tooltip that names every tag.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { measureRect, Tooltip, type Box } from '@numen/ui'
import { getTagColor } from '../../lib/color'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ tags: readonly string[] }>()

/* --------------------------------- State ---------------------------------- */
const DOT_LIMIT = 4
const DOTS_BEFORE_MORE = 3

const root = useTemplateRef<HTMLElement>('root')
const at = ref<Box | null>(null)

const hasMore = computed(() => props.tags.length > DOT_LIMIT)
const dotCount = computed(() => (hasMore.value ? DOTS_BEFORE_MORE : props.tags.length))
const dots = computed(() =>
  props.tags.slice(0, dotCount.value).map((tag) => ({
    tag,
    style: { '--tag': `var(--numen-tag-${getTagColor(tag)})` },
  })),
)
const more = computed(() => props.tags.length - DOTS_BEFORE_MORE)
const moreSize = computed(() =>
  more.value >= 100 ? 'file-tags__more--wider' : more.value >= 10 ? 'file-tags__more--wide' : '',
)
const label = computed(() => props.tags.join(', '))

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
      <span v-if="hasMore" class="file-tags__dot file-tags__more" :class="moreSize" data-tag-more>
        +{{ more }}
      </span>
    </span>
    <Teleport to="body">
      <Tooltip v-if="at" :at="at">{{ label }}</Tooltip>
    </Teleport>
  </template>
</template>

<style scoped>
.file-tags {
  --ring: 0.0625rem;
  --dot: 0.625rem;

  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  margin-inline-start: auto;
  font-size: var(--numen-text-1);
  line-height: 1;
}

.file-tags__dot {
  box-sizing: content-box;
  inline-size: var(--dot);
  block-size: var(--dot);
  border-radius: 50%;
  background: var(--tag);
  box-shadow: 0 0 0 var(--ring) var(--tree-row-ring, var(--tree-row-bg));
}

.file-tags__dot + .file-tags__dot {
  margin-inline-start: calc(var(--dot) * -0.3);
}

.file-tags__more {
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  --dot: 0.875rem;
  background: var(--numen-rule);
  color: var(--numen-ink);
  font-size: 0.5625rem;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.file-tags__more--wide {
  font-size: 0.5rem;
}

.file-tags__more--wider {
  font-size: 0.4375rem;
}
</style>
