<script setup lang="ts">
/**
 * The tab of a file whose bytes are being brought onto this device: a ring with the file name while
 * it downloads, the failure and a retry once it fails.
 */
import { computed } from 'vue'
import { RefreshCw, TriangleAlert } from '@lucide/vue'
import { Button, Spinner } from '@numen/ui'
import type { DownloadState } from '../model/createDownloadTabs'
import { WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: DownloadState
}>()

/* --------------------------------- State ---------------------------------- */
const view = computed(() => props.state.view.value)
const isFailed = computed(() => view.value.failure !== '')
</script>

<template>
  <div class="download-pane" role="status" :aria-busy="!isFailed">
    <div class="download-pane__content">
      <TriangleAlert v-if="isFailed" class="download-pane__icon" aria-hidden="true" />
      <span v-else class="download-pane__ring">
        <Spinner class="download-pane__spinner" />
        <RefreshCw class="download-pane__arrows" aria-hidden="true" />
      </span>
      <h3 class="download-pane__title">{{ isFailed ? view.failure : WORDS.downloading }}</h3>
      <p class="download-pane__text">{{ view.title }}</p>
      <Button v-if="isFailed" variant="ghost" size="small" @click="state.retry()">
        {{ WORDS.retry }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.download-pane {
  --pad: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  block-size: 100%;
  padding: var(--pad);
  box-sizing: border-box;
}

.download-pane__content {
  --max-width: 28rem;
  --gap: 0.5rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--gap);
  text-align: center;
  max-inline-size: var(--max-width);
}

.download-pane__ring {
  --size: 3rem;
  position: relative;
  display: grid;
  place-items: center;
  inline-size: var(--size);
  block-size: var(--size);
  color: var(--numen-hushed);
}

.download-pane__spinner {
  --spinner-size: 3rem;
  --spinner-thickness: 0.125rem;
  position: absolute;
  inset: 0;
}

.download-pane__arrows {
  --size: 1.25rem;
  inline-size: var(--size);
  block-size: var(--size);
}

.download-pane__icon {
  --size: 2.5rem;
  inline-size: var(--size);
  block-size: var(--size);
  color: var(--numen-caution-fg);
}

.download-pane__title {
  margin: 0;
  font-size: var(--numen-text-4);
  font-weight: 600;
  color: var(--numen-ink);
  overflow-wrap: anywhere;
}

.download-pane__text {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  line-height: 1.4;
  overflow-wrap: anywhere;
}
</style>
