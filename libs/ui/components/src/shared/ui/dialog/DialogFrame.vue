<script setup lang="ts">
/** A window in miniature for stories and tests: a title bar, the content that hosts dialogs, a status bar. */
import { useTemplateRef } from 'vue'
import { provideDialogHost } from './host'

const content = useTemplateRef<HTMLElement>('content')
provideDialogHost(content)
</script>

<template>
  <div class="frame">
    <header data-slot="title-bar" class="frame__bar">Title bar</header>
    <main ref="content" data-slot="content" class="frame__content"><slot /></main>
    <footer data-slot="status-bar" class="frame__bar">Status bar</footer>
  </div>
</template>

<style scoped>
.frame {
  display: flex;
  flex-direction: column;
  inline-size: 60rem;
  block-size: 30rem;
  background: var(--numen-surface);
  color: var(--numen-ink);
}

.frame__bar {
  flex: none;
  padding: 0.5rem 1rem;
  background: var(--numen-raised);
}

.frame__content {
  position: relative;
  flex: 1;
  min-block-size: 0;
}
</style>
