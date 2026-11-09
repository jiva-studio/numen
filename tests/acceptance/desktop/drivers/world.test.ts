import fs from 'node:fs'
import path from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'
import {
  assertVaultName,
  assertVaultPath,
  DEFAULT_VAULT,
  getNoteFile,
  parseVaultName,
  pickVault,
} from './rules.js'
import { World } from './world.js'

describe('parseVaultName', () => {
  it('gives the default for the vault without a name, and the name for the vault with one', () => {
    expect(parseVaultName('the vault')).toBe(DEFAULT_VAULT)
    expect(parseVaultName('the vault "Archive"')).toBe('Archive')
  })

  it('throws for words that are not a vault', () => {
    expect(() => parseVaultName('the vaults')).toThrow('does not name a vault')
  })
})

describe('pickVault', () => {
  it('takes the only vault when none is named', () => {
    expect(pickVault(['Test'], DEFAULT_VAULT)).toBe('Test')
  })

  it('takes the vault that is named', () => {
    expect(pickVault(['Test', 'Archive'], 'Archive')).toBe('Archive')
  })

  it('asks for a name when the world has two vaults', () => {
    expect(() => pickVault(['Test', 'Archive'], DEFAULT_VAULT)).toThrow('name the vault: the world has Test and Archive')
  })

  it('asks for a vault when the world has none', () => {
    expect(() => pickVault([], DEFAULT_VAULT)).toThrow('the world has no vault')
  })

  it('throws for a vault the world does not have', () => {
    expect(() => pickVault(['Test'], 'Archive')).toThrow('the world has no vault "Archive": it has Test')
  })
})

describe('assertVaultName', () => {
  it('takes a plain name', () => {
    expect(() => assertVaultName('My Archive')).not.toThrow()
  })

  it.each(['', DEFAULT_VAULT, '.hidden', 'a/b', 'a\\b', 'a:b'])('throws for "%s"', (name) => {
    expect(() => assertVaultName(name)).toThrow('cannot name a vault')
  })
})

describe('getNoteFile', () => {
  it('puts a note in the root of the vault, named by its title', () => {
    expect(getNoteFile('Entropy')).toBe('Entropy.md')
    expect(getNoteFile(' Mill race ')).toBe('Mill race.md')
  })

  it.each(['', '  ', 'a/b', 'a#b', 'a:b'])('throws for the title "%s"', (title) => {
    expect(() => getNoteFile(title)).toThrow('cannot be the title of a note file')
  })
})

describe('assertVaultPath', () => {
  it('takes a relative path', () => {
    expect(() => assertVaultPath('assets/shore.pdf')).not.toThrow()
  })

  it.each(['/etc/passwd', '../outside', 'a/../../b', 'a//b', './a', ''])('throws for "%s"', (entryPath) => {
    expect(() => assertVaultPath(entryPath)).toThrow('is not a path inside the vault')
  })
})

describe('World', () => {
  const worlds: World[] = []
  const create = (): World => {
    const world = World.create()
    worlds.push(world)
    return world
  }
  afterEach(() => {
    for (const world of worlds.splice(0)) world.remove()
  })

  it('keeps one folder for each vault, named by the vault', () => {
    const world = create()
    world.addVault('Test')
    world.addVault('Archive')
    expect(world.getVaults()).toEqual(['Test', 'Archive'])
    expect(world.getFolder('Archive')).toBe(path.join(world.root, 'Archive'))
    expect(fs.statSync(world.getFolder('Test')).isDirectory()).toBe(true)
  })

  it('refuses the same vault twice', () => {
    const world = create()
    world.addVault('Test')
    expect(() => world.addVault('Test')).toThrow('has the vault "Test" already')
  })

  it('needs a name for the vault when it has two', () => {
    const world = create()
    world.addVault('Test')
    world.addVault('Archive')
    expect(() => world.getFolder(DEFAULT_VAULT)).toThrow('name the vault: the world has Test and Archive')
  })

  it('writes a note into the root of the vault', () => {
    const world = create()
    world.addVault('Test')
    world.writeNote(DEFAULT_VAULT, 'Entropy', 'A measure of uncertainty.')
    expect(fs.readFileSync(path.join(world.getFolder('Test'), 'Entropy.md'), 'utf8')).toBe('A measure of uncertainty.\n')
    expect(world.getEntryKind('Test', 'Entropy.md')).toBe('file')
  })

  it('makes a folder and copies a PDF from the samples under its own name', () => {
    const world = create()
    world.addVault('Test')
    world.makeFolder('Test', 'daily/2026')
    world.copyPdf('Test', 'assets/shore.pdf')
    expect(world.getEntryKind('Test', 'daily/2026')).toBe('folder')
    expect(world.getEntryKind('Test', 'assets/shore.pdf')).toBe('file')
    expect(world.getEntryKind('Test', 'assets/other.pdf')).toBeUndefined()
  })

  it('has no sample for a PDF the library does not know', () => {
    const world = create()
    world.addVault('Test')
    expect(() => world.copyPdf('Test', 'assets/unknown.pdf')).toThrow('there is no sample PDF "unknown.pdf"')
  })

  it('keeps every vault apart', () => {
    const world = create()
    world.addVault('Test')
    world.addVault('Archive')
    world.writeNote('Archive', 'Old', 'Kept.')
    expect(world.getEntryKind('Archive', 'Old.md')).toBe('file')
    expect(world.getEntryKind('Test', 'Old.md')).toBeUndefined()
  })

  it('does not leave the vault', () => {
    const world = create()
    world.addVault('Test')
    expect(() => world.makeFolder('Test', '../outside')).toThrow('is not a path inside the vault')
  })

  it('removes the folder of the world with every vault', () => {
    const world = create()
    world.addVault('Test')
    world.remove()
    expect(fs.existsSync(world.root)).toBe(false)
  })
})
