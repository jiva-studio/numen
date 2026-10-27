<script setup lang="ts">
/**
 * The way to add a template, on a rule at the end of the templates already cut.
 */
import { Button } from '@numen/ui'
import { Icon } from '../../icon'
import { Divider } from '../../divider'

defineProps<{
  /** What the button says, and is announced as. */
  label: string
}>()

const emit = defineEmits<{
  (event: 'press'): void
}>()
</script>

<template>
  <Divider>
    <Button variant="ghost" size="small" @click="emit('press')">
      <Icon name="plus" />
      {{ label }}
    </Button>
  </Divider>
</template>
