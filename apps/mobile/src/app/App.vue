<script setup lang="ts">
/** The application: one page, showing the vault the core opened and the files of its whole tree. */
import { IonApp } from '@ionic/vue'
import { FileTreePage } from '@/pages/file-tree'
import { NoteSheet } from '@/pages/note-sheet'
import { VaultPage } from '@/pages/vault'
</script>

<template>
  <IonApp>
    <VaultPage>
      <template #tree="{ open }">
        <FileTreePage @open="open" />
      </template>
      <template #note="{ core, path, close, report }">
        <NoteSheet :core="core" :path="path" @close="close" @error="report" />
      </template>
    </VaultPage>
  </IonApp>
</template>
