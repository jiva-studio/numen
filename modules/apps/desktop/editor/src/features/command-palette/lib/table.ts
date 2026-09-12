/**
 * Construction of the full list of commands offered by the application.
 */
import { keysOf } from './chords'
import { getWhereOnEvidence, isOnAnything, isOnNote, isOnVault, isUnmade } from './where'
import type { Command } from '../types'
import type { Words } from '../words'

/**
 * Every command, in the order it is drawn. The keyboard it is being read on
 * decides how the keystrokes on it are written.
 *
 * A command reached by Shift and Enter on another one's row is offered here
 * too and drawn nowhere: the row it belongs to is the one that names it.
 */
export const commandsOf = (
  words: Words,
  agent: string = navigator.userAgent,
): readonly Command[] => [
  { id: 'read', text: words.read, group: 'note', where: isOnNote, also: 'beside' },
  { id: 'beside', text: words.beside, group: 'note', where: isOnNote },
  { id: 'travel', text: words.travel, ...keysOf('travel', agent), group: 'note', where: isOnNote },
  {
    id: 'child',
    text: words.child,
    ...keysOf('child', agent),
    group: 'note',
    needs: 'naming',
    where: isOnNote,
  },
  { id: 'parent', text: words.parent, group: 'note', needs: 'naming', where: isOnNote },
  { id: 'jump', text: words.jump, group: 'note', needs: 'naming', where: isOnNote },
  {
    id: 'title',
    text: words.title,
    group: 'note',
    needs: 'naming',
    where: isOnNote,
    getFieldText: (at) => at.title,
  },
  // The note goes to the vault's .trash folder, and putting it back is a move.
  // Destroy is the one that asks.
  { id: 'remove', text: words.remove, group: 'note', where: isOnNote, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    where: isOnNote,
    warns: { action: words.destroys, then: words.forever, back: words.typeBack },
  },
  { id: 'ask', text: words.ask, group: 'note', where: isOnNote },
  { id: 'copy', text: words.copy, group: 'note', where: isOnNote },
  { id: 'reveal', text: words.reveal, group: 'note', where: isOnNote },
  { id: 'preset', text: words.preset, group: 'note', where: isOnNote },
  {
    id: 'transcribe',
    text: words.transcribe,
    group: 'file',
    where: getWhereOnEvidence('transcribe', 'recording', (made) => isUnmade(made.transcript)),
  },
  {
    id: 'downloadText',
    text: words.downloadText,
    group: 'file',
    where: getWhereOnEvidence(
      'downloadText',
      'url',
      (made) => made.transcript !== undefined || made.article !== undefined,
    ),
  },
  {
    id: 'downloadCopy',
    text: words.downloadCopy,
    group: 'file',
    // Only a note pointing at a video carries a copy at all, so the row being
    // there is what says this file is one. An hour of video on somebody's disk
    // is asked for by hand, and one already here is not asked for again.
    where: (at, runs) =>
      at.ready &&
      runs.canRun('downloadCopy') &&
      at.made.copy !== undefined &&
      isUnmade(at.made.copy),
  },
  {
    id: 'proofread',
    text: words.proofread,
    group: 'file',
    // There is nothing to put right until a model has heard something, and
    // nothing to put right again once it has been put right.
    where: getWhereOnEvidence(
      'proofread',
      'recording',
      (made) => made.transcript === 'done' && isUnmade(made['transcript.corrected']),
    ),
  },
  {
    id: 'deleteText',
    text: words.deleteText,
    group: 'file',
    needs: 'asking',
    // Everything one run of listening left goes, so a run that stopped part way
    // and a recording that gave no words are both taken away here. A url
    // carries a transcript the same way, and deletes it here.
    where: (at, runs) =>
      getWhereOnEvidence('deleteText', 'recording', (made) => made.transcript !== 'none')(at, runs) ||
      getWhereOnEvidence('deleteText', 'url', (made) => made.transcript !== 'none')(at, runs),
    answers: {
      keeps: words.keepsTranscript,
      kept: words.kept,
      action: words.deletes,
      then: words.deleted,
    },
  },
  {
    id: 'deleteCopy',
    text: words.deleteCopy,
    group: 'file',
    needs: 'asking',
    // Only a url carries a copy, and only one that stands has anything to take.
    where: getWhereOnEvidence('deleteCopy', 'url', (made) => made.copy === 'done'),
    answers: {
      keeps: words.keepsCopy,
      kept: words.kept,
      action: words.deletes,
      then: words.deletedCopy,
    },
  },
  {
    id: 'recognise',
    text: words.recognise,
    group: 'file',
    where: getWhereOnEvidence('recognise', 'book', (made) => isUnmade(made.ocr)),
  },
  {
    id: 'note',
    text: words.newNote,
    ...keysOf('note', agent),
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'deck',
    text: words.newDeck,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'importUrl',
    text: words.importUrl,
    group: 'window',
    needs: 'address',
    where: (at) => at.ready,
  },
  { id: 'plex', text: words.newPlex, ...keysOf('plex', agent), group: 'window', where: isOnAnything },
  { id: 'files', text: words.files, group: 'window', where: isOnAnything },
  { id: 'agent', text: words.newAgent, ...keysOf('agent', agent), group: 'window', where: isOnAnything },
  {
    id: 'close',
    text: words.close,
    ...keysOf('close', agent),
    group: 'window',
    where: (at) => at.tab !== '',
  },
  { id: 'find', text: words.find, keys: words.findKeys, group: 'window', where: isOnAnything },
  { id: 'appearance', text: words.appearance, group: 'window', needs: 'choosing', where: isOnAnything },
  { id: 'mode', text: words.mode, group: 'window', needs: 'choosing', where: isOnAnything },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'window',
    needs: 'choosing',
    where: isOnAnything,
  },
  { id: 'textScale', text: words.textScale, group: 'window', needs: 'choosing', where: isOnAnything },
  { id: 'syncing', text: words.syncing, group: 'window', needs: 'choosing', where: isOnAnything },
  { id: 'hanging', text: words.hanging, group: 'window', needs: 'choosing', where: isOnAnything },
  { id: 'parts', text: words.parts, group: 'window', needs: 'choosing', where: isOnAnything },
  { id: 'settings', text: words.settings, group: 'window', where: isOnAnything },
  { id: 'first', text: words.first, group: 'vault', where: (at) => at.ready },
  {
    id: 'goto',
    text: words.goto,
    ...keysOf('goto', agent),
    group: 'vault',
    needs: 'picking',
    where: (at) => at.ready,
  },
  { id: 'openVault', text: words.openVault, group: 'vault', needs: 'vaults', where: isOnAnything },
  {
    id: 'newVault',
    text: words.newVault,
    ...keysOf('newVault', agent),
    group: 'vault',
    where: isOnAnything,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    group: 'vault',
    needs: 'naming',
    where: isOnVault,
    getFieldText: (at) => at.vault.name,
  },
  {
    id: 'forgetVault',
    text: words.forgetVault,
    group: 'vault',
    needs: 'vaults',
    next: 'asking',
    where: isOnAnything,
    answers: {
      keeps: words.keepsVault,
      kept: words.kept,
      action: words.forgets,
      then: words.stays,
    },
  },
  {
    id: 'eraseVault',
    text: words.eraseVault,
    group: 'vault',
    needs: 'vaults',
    next: 'exactly',
    where: isOnAnything,
    warns: { action: words.erases, then: words.binned, back: words.typeVaultBack },
  },
]
