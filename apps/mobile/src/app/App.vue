<script setup lang="ts">
/** The application: one page, showing the vault the core opened. */
import { IonApp } from '@ionic/vue'
import { NoteSheet } from '@/pages/note-sheet'
import { VaultPage } from '@/pages/vault'
</script>

<template>
  <IonApp>
    <VaultPage>
      <template #note="{ core, path, close, report }">
        <NoteSheet :core="core" :path="path" @close="close" @error="report" />
      </template>
    </VaultPage>
  </IonApp>
</template>
