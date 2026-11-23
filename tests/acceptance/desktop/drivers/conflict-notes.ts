/** The notes Numen Sync makes of the text a folder held when its note met a synced note with another text. */

const escapePattern = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** The texts of the conflict notes of a note: the files named "Title (conflict 2026-11-23).md" in its folder, with a number after the day once that name is taken. */
export const findConflictNotes = (files: readonly { path: string; text: string }[], title: string): string[] => {
  const pattern = new RegExp(`^${escapePattern(title)} \\(conflict \\d{4}-\\d{2}-\\d{2}(?: \\d+)?\\)\\.md$`)
  return files.filter((file) => pattern.test(file.path)).map((file) => file.text.trimEnd())
}
