import assert from 'node:assert/strict'
import test from 'node:test'
import { clauses, code, silent } from './catches.mjs'
import { sources } from './source.mjs'

/**
 * A swallowed failure is invisible everywhere a fault is normally met. The work
 * does not happen, the window says nothing, and every test and every story goes
 * on passing, because none of them can assert an absence they were never told
 * to expect. Only the person meets it, and they meet it as a screen that sits
 * there looking fine.
 *
 * The catch that reaches a person is not the problem; the window has a voice
 * and those catches use it. The problem is the one that reaches nobody, and
 * nothing but a line of writing tells a deliberate one from a forgotten one.
 */
test('a catch that discards the error says why', () => {
  const wrong = []
  const read = []
  for (const { at, text } of sources(['.ts', '.vue'])) {
    const parts = code(at, text)
    if (!parts.some((part) => clauses(part).length > 0)) continue
    read.push(at)
    for (const part of parts) {
      for (const line of silent(part)) {
        wrong.push(`${at}:${line} catches and says nothing about what it caught`)
      }
    }
  }
  assert.deepEqual(wrong, [])

  // A walk that read no catch is a rule checked against nothing, and it passes.
  // The count is a floor well under what the modules hold, and two files are
  // named besides: the one plainest instance the sweep found, and the phone,
  // which is the border a walk reading the wrong tree stops at.
  assert.ok(
    read.length > 25,
    `${read.length} files holding a catch read: the walk is not reading them`,
  )
  assert.ok(
    read.some((at) => at.endsWith('desktop/ui/src/note/naming.ts')),
    'the walk did not read naming.ts, whose bare catch is what this rule was written for',
  )
  assert.ok(
    read.some((at) => at.endsWith('apps/mobile/src/plex/following.ts')),
    "the walk did not read the phone's following.ts, so the rule stops at the mobile border",
  )
})

/**
 * What the rule refuses, read against clauses written to be refused. What it
 * lets through is a catch that does something with the error and a catch that
 * says why there is nothing to do — never a count of exceptions.
 */
test('what the catch rule refuses', () => {
  const cases = [
    {
      says: 'a bare catch',
      allowed: false,
      source: 'try { open() } catch {}',
    },
    {
      says: 'a catch that only returns',
      allowed: false,
      source: 'try { return read() } catch { return null }',
    },
    {
      says: 'a catch binding an error it never names',
      allowed: false,
      source: 'try { open() } catch (why) { shut() }',
    },
    {
      says: 'a catch saying why there was nothing to do',
      allowed: true,
      source: 'try { open() } catch {\n  // Never held it; nothing to let go of.\n}',
    },
    {
      says: 'a catch that tells the person what went wrong',
      allowed: true,
      source: 'try { open() } catch (why) { said(troubleWords(why), "refusal") }',
    },
    {
      says: 'a catch that puts the error back',
      allowed: true,
      source: 'try { open() } catch (why) { throw why }',
    },
    {
      says: 'a catch naming the error inside a template literal',
      allowed: true,
      source: 'try { open() } catch (why) { wrong.value = `unread ${troubleWords(why)}` }',
    },
    {
      says: 'a catch carrying a block comment',
      allowed: true,
      source: 'try { open() } catch {\n  /* The next change asks again. */\n}',
    },
    {
      says: 'a promise handler, which is not a clause and is read where it stands',
      allowed: true,
      source: 'void open().catch(() => {})',
    },
    {
      says: 'the word catch inside a string',
      allowed: true,
      source: 'const words = { follow: "nothing to catch {} here" }',
    },
    {
      says: 'the word catch inside a comment',
      allowed: true,
      source: '// A handle that catches { its whole reach } and no further.\nconst a = 1',
    },
    {
      says: 'a bare catch inside a nested block',
      allowed: false,
      source: 'if (open) { for (;;) { try { read() } catch { break } } }',
    },
  ]

  const refused = cases.filter((one) => silent(one.source).length > 0).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(refused, wanted)
})
