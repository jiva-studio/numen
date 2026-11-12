/**
 * What the tests of decks, recordings, links, the agent and the updater need beyond `vault.ts`:
 * files copied and removed on disk, the folders plugins keep in the vault, deck files, and the
 * stand-ins' address.
 */
import fs from 'node:fs'
import path from 'node:path'
import zlib from 'node:zlib'
import { expect, type APIRequestContext } from 'playwright/test'
import { getVaultPath, listFilePaths, readOwner, writeSetting } from './vault.js'
import type { JsonValue } from './json.js'

/** Whether a file of the vault is on disk. */
export const hasVaultFile = (file: string): boolean =>
  fs.existsSync(path.join(getVaultPath(), file))

/** Copies a file of the vault to another path of it. */
export const copyVaultFile = (from: string, to: string): void => {
  const root = getVaultPath()
  fs.copyFileSync(path.join(root, from), path.join(root, to))
}

/** Deletes files of the vault from the disk; a missing one is skipped. */
export const deleteVaultFiles = (...files: string[]): void => {
  const root = getVaultPath()
  for (const file of files) fs.rmSync(path.join(root, file), { force: true })
}

/** The names of the files a plugin keeps in its folder of the vault, `.numen/<plugin id>/`. */
export const listPluginFiles = (plugin: string): string[] => {
  const folder = path.join(getVaultPath(), '.numen', plugin)
  return fs.existsSync(folder) ? fs.readdirSync(folder) : []
}

/** Waits until the core lists the file in its folder. */
export const waitForListedFile = async (
  request: APIRequestContext,
  file: string,
): Promise<void> => {
  const folder = path.posix.dirname(file) === '.' ? '' : path.posix.dirname(file)
  await expect.poll(() => listFilePaths(request, folder)).toContain(file)
}

/** The letters a card id of a deck is written in (the idPattern of the flashcards manifest). */
const CARD_ID_LETTERS = '0123456789abcdefghjkmnpqrstvwxyz'

/** A card id no fixture of the vault carries. */
export const createCardId = (): string =>
  Array.from(
    { length: 10 },
    () => CARD_ID_LETTERS[Math.floor(Math.random() * CARD_ID_LETTERS.length)],
  ).join('')

/** One card of a deck file: its id, the card model that cuts it and its values by field. */
export interface DeckCard {
  readonly id: string
  readonly cardModel: string
  readonly values: Readonly<Record<string, string>>
}

/** The text of a deck file holding the cards under its title, before any section. */
export const createDeckText = (title: string, cards: readonly DeckCard[]): string =>
  [
    `---\ntype: deck\ntitle: ${title}\n---`,
    ...cards.map((card) =>
      [
        `## ${Object.values(card.values)[0] ?? ''} ^${card.id}`,
        `[[${card.cardModel}]]`,
        ...Object.entries(card.values).map(([field, value]) => `### ${field}\n\n${value}`),
      ].join('\n\n'),
    ),
  ].join('\n\n') + '\n'

/** Writes one setting of a plugin into the run's settings file. */
export const writePluginSetting = (
  request: APIRequestContext,
  plugin: string,
  key: string,
  value: JsonValue | undefined,
): Promise<void> => writeSetting(request, [plugin, key], value)

/**
 * Stops a plugin and starts it again, which takes the tasks it reported out of the list: the core
 * lists a failed task until the plugin ends it or stops.
 */
export const restartPlugin = async (request: APIRequestContext, plugin: string): Promise<void> => {
  await writePluginSetting(request, plugin, 'enabled', false)
  await writePluginSetting(request, plugin, 'enabled', true)
}

/**
 * The origin of the stand-ins `scripts/stand-ins.mjs` starts, read from the release feed address
 * the run's settings give the updater plugin.
 */
export const getStandInsOrigin = async (request: APIRequestContext): Promise<string> => {
  const endpoint = String((await readOwner(request, 'md.numen.tools.updater'))['endpoint'] ?? '')
  if (!endpoint) throw new Error('the settings give the updater no release feed address')
  return new URL(endpoint).origin
}

/** One chapter of a book: its title in the contents and the paragraph it holds. */
export interface Chapter {
  readonly title: string
  readonly text: string
}

/** One file of a zip archive. */
interface ArchiveEntry {
  readonly name: string
  readonly content: string
}

const createArchive = (entries: readonly ArchiveEntry[]): Buffer => {
  const parts: Buffer[] = []
  const directory: Buffer[] = []
  let offset = 0
  for (const entry of entries) {
    const name = Buffer.from(entry.name)
    const content = Buffer.from(entry.content)
    const header = Buffer.alloc(30)
    header.writeUInt32LE(0x04034b50, 0)
    header.writeUInt16LE(20, 4)
    header.writeUInt32LE(zlib.crc32(content), 14)
    header.writeUInt32LE(content.length, 18)
    header.writeUInt32LE(content.length, 22)
    header.writeUInt16LE(name.length, 26)
    const record = Buffer.alloc(46)
    record.writeUInt32LE(0x02014b50, 0)
    record.writeUInt16LE(20, 4)
    record.writeUInt16LE(20, 6)
    record.writeUInt32LE(zlib.crc32(content), 16)
    record.writeUInt32LE(content.length, 20)
    record.writeUInt32LE(content.length, 24)
    record.writeUInt16LE(name.length, 28)
    record.writeUInt32LE(offset, 42)
    directory.push(record, name)
    parts.push(header, name, content)
    offset += header.length + name.length + content.length
  }
  const end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50, 0)
  end.writeUInt16LE(entries.length, 8)
  end.writeUInt16LE(entries.length, 10)
  end.writeUInt32LE(
    directory.reduce((sum, one) => sum + one.length, 0),
    12,
  )
  end.writeUInt32LE(offset, 16)
  return Buffer.concat([...parts, ...directory, end])
}

/** An EPUB 3 book of the chapters, each in a document of its own and listed in its contents. */
export const createEpub = (title: string, chapters: readonly Chapter[]): Buffer => {
  const page = (body: string): string =>
    `<?xml version="1.0" encoding="UTF-8"?>\n<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"><head><title>${title}</title></head><body>${body}</body></html>`
  return createArchive([
    { name: 'mimetype', content: 'application/epub+zip' },
    {
      name: 'META-INF/container.xml',
      content:
        '<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>',
    },
    {
      name: 'OEBPS/content.opf',
      content: `<?xml version="1.0" encoding="UTF-8"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>${title}</dc:title><dc:identifier id="id">urn:uuid:${title}</dc:identifier><dc:language>en</dc:language></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>${chapters
        .map(
          (_, index) =>
            `<item id="c${index}" href="c${index}.xhtml" media-type="application/xhtml+xml"/>`,
        )
        .join(
          '',
        )}</manifest><spine>${chapters.map((_, index) => `<itemref idref="c${index}"/>`).join('')}</spine></package>`,
    },
    {
      name: 'OEBPS/nav.xhtml',
      content: page(
        `<nav epub:type="toc"><ol>${chapters
          .map((one, index) => `<li><a href="c${index}.xhtml">${one.title}</a></li>`)
          .join('')}</ol></nav>`,
      ),
    },
    ...chapters.map((one, index) => ({
      name: `OEBPS/c${index}.xhtml`,
      content: page(`<h1>${one.title}</h1><p>${one.text}</p>`),
    })),
  ])
}

/** Writes bytes into a file of the vault as a program outside numen would. */
export const writeVaultBytes = (file: string, bytes: Buffer): void => {
  fs.writeFileSync(path.join(getVaultPath(), file), bytes)
}
