<script setup lang="ts">
import { LoadingIndicator } from '@numen/ui'
import { WORDS as words } from '../../../../words'
</script>

<template>
  <LoadingIndicator data-control="waiting" :label="words.waiting" :delay="0" />
</template>
