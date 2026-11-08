<script setup lang="ts">
/**
 * The icon button at the end of a row or a card header, drawn with the corner radius of the fields
 * beside it. Attributes and listeners fall through to the button.
 */
import { Button } from '@numen/ui'
</script>

<template>
  <Button variant="outline" size="icon" class="secret-action">
    <slot />
  </Button>
</template>

<style scoped>
.secret-action {
  border-radius: var(--numen-radius);
}
</style>
