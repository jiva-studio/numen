<script setup lang="ts">
/**
 * What stands in the curve's room while the answer that draws it is read.
 */
import { Waiting } from '@numen/ui'
import { WORDS as words } from '../../../../words'
</script>

<template>
  <Waiting data-control="waiting" :label="words.waiting" />
</template>
