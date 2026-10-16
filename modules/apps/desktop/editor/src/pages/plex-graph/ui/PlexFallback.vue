<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
defineProps<{
  path?: string
}>()
</script>

<template>
  <div class="plex-fallback">
    <div class="plex-fallback__card">
      <div class="plex-fallback__badge">Plugin Required</div>
      <h2 class="plex-fallback__title">Knowledge Graph Visualizer</h2>
      <p class="plex-fallback__message">
        Plexus Graph visualizer plugin (md.numen.view-plex-graph) is not active. Install or enable the Plex Graph
        plugin to view and navigate the knowledge graph.
      </p>
    </div>
  </div>
</template>

<style scoped>
.plex-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  block-size: 100%;
  padding: 2rem;
  background-color: var(--color-background, #fff);
  color: var(--color-foreground, #000);
}

.plex-fallback__card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-inline-size: 28rem;
  padding: 2.5rem 2rem;
  border: 1px solid var(--color-edge, #e5e7eb);
  border-radius: 8px;
  background-color: var(--color-surface, #f9fafb);
}

.plex-fallback__badge {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  background-color: var(--color-warning-subtle, #fef3c7);
  color: var(--color-warning, #b45309);
  margin-bottom: 1rem;
}

.plex-fallback__title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 0.75rem;
  word-break: break-word;
}

.plex-fallback__message {
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--color-muted, #6b7280);
  margin: 0;
}
</style>
