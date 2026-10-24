<script setup lang="ts">
import { Waiting } from '@numen/ui'
import { WORDS as words } from '../../../../words'
</script>

<template>
  <Waiting data-control="waiting" :label="words.waiting" />
</template>
