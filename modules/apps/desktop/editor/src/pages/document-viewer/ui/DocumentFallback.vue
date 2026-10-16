<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  path: string
}>()
</script>

<template>
  <div class="document-fallback">
    <div class="document-fallback__card">
      <div class="document-fallback__badge">Plugin Required</div>
      <h2 class="document-fallback__title">{{ props.path.split('/').pop() ?? props.path }}</h2>
      <p class="document-fallback__message">
        PDF reader plugin (md.numen.pdf) is not active. Install or enable the PDF reader
        plugin to view PDF documents.
      </p>
    </div>
  </div>
</template>

<style scoped>
.document-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  block-size: 100%;
  padding: 2rem;
  background-color: var(--color-background, #fff);
  color: var(--color-foreground, #000);
}

.document-fallback__card {
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

.document-fallback__badge {
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

.document-fallback__title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 0.75rem;
  word-break: break-word;
}

.document-fallback__message {
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--color-muted, #6b7280);
  margin: 0;
}
</style>
