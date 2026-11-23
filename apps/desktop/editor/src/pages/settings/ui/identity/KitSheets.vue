<script setup lang="ts">
/**
 * The three sheets of a recovery kit as they are printed: one sheet to a page, each with its
 * number, the kit, the account, the 33 numbered words and what to do with the sheet.
 */
import type { RecoveryKit } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ kit: RecoveryKit }>()
</script>

<template>
  <div class="kit-sheets">
    <article
      v-for="(sheet, index) in kit.sheets"
      :key="index"
      class="kit-sheet"
      :aria-label="words.kitSheetLabel(index + 1)"
    >
      <h3 class="kit-sheet__title">{{ words.kitSheetTitle }}</h3>
      <p class="kit-sheet__line">{{ words.kitSheetLabel(index + 1) }}</p>
      <p class="kit-sheet__line">{{ words.kitSheetKit(kit.id) }}</p>
      <p class="kit-sheet__line">
        {{ words.kitSheetAccount }}
        <span class="kit-sheet__mono">{{ kit.accountFingerprint }}</span>
      </p>
      <ol class="kit-sheet__words">
        <li v-for="(word, at) in sheet" :key="at" class="kit-sheet__word">{{ word }}</li>
      </ol>
      <p class="kit-sheet__note">{{ words.kitSheetRule }}</p>
      <p class="kit-sheet__note">{{ words.kitSheetWarning }}</p>
    </article>
  </div>
</template>

<style scoped>
.kit-sheets {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.kit-sheet {
  padding: 0.75rem;
  background-color: var(--numen-bubble-bg);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
}

.kit-sheet__title {
  margin: 0 0 0.25rem;
  font-size: var(--numen-text-3);
  font-weight: 600;
}

.kit-sheet__line {
  margin: 0;
}

.kit-sheet__mono {
  font-family: var(--numen-font-mono);
}

.kit-sheet__words {
  columns: 3;
  margin: 0.75rem 0;
  padding-inline-start: 2.5rem;
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.kit-sheet__word {
  padding-inline-start: 0.25rem;
  break-inside: avoid;
}

.kit-sheet__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>

<style>
@media print {
  body * {
    visibility: hidden;
  }

  .kit-sheets,
  .kit-sheets * {
    visibility: visible;
  }

  .kit-sheets {
    position: absolute;
    inset-block-start: 0;
    inset-inline-start: 0;
    inline-size: 100%;
    display: block;
  }

  .kit-sheet {
    break-after: page;
  }
}
</style>
