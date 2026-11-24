import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { findNativeTitles } from './check.mjs'

const vue = (template) => `<script setup lang="ts"></script>\n<template>\n${template}\n</template>`

describe('native title attributes', () => {
  it('flags a static, bound or multi-line title on an HTML element', () => {
    assert.deepEqual(findNativeTitles(vue('  <button title="Save">x</button>')), [3])
    assert.deepEqual(findNativeTitles(vue('  <span :title="value" />')), [3])
    assert.deepEqual(findNativeTitles(vue('  <button\n    type="button"\n    v-bind:title="label"\n  >x</button>')), [3])
  })

  it('leaves the title prop of a component and the aria-label alone', () => {
    assert.deepEqual(findNativeTitles(vue('  <IslandCard :title="words.card" aria-label="x" />')), [])
    assert.deepEqual(findNativeTitles(vue('  <button aria-label="Save" data-title="x">x</button>')), [])
  })

  it('leaves the text and an SVG title element alone', () => {
    assert.deepEqual(findNativeTitles(vue('  <svg><title>Logo</title></svg>\n  <p>a title="x" in text</p>')), [])
  })
})
