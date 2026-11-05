#!/usr/bin/env node
/**
 * The plugins the macOS application bundle copies for a target.
 *
 *   node select.mjs --target <target> [--plugins <dir>]
 *     prints the folder name of each plugin to copy, one per line, ordered by name
 *   node select.mjs --target <target> [--plugins <dir>] --verify <dir>
 *     exits non-zero and names each folder when the plugins index lists a plugin for the target
 *     that <dir> (the plugins folder of the application) does not hold
 *
 * Contract frozen for the tests in select.test.mjs; the implementation is not written yet.
 */
