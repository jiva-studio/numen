import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TreeField from './TreeField.vue'

describe('TreeField', () => {
  it('selects only the stem for a file with an extension', () => {
    const wrapper = mount(TreeField, {
      attachTo: document.body,
      props: {
        value: 'component.vue',
        name: 'File name',
        isFolder: false,
      },
    })

    wrapper.vm.focus()
    const input = wrapper.find<HTMLInputElement>('input').element
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe(9) // 'component'
    wrapper.unmount()
  })

  it('selects the whole name up to the last dot for multi-extension files', () => {
    const wrapper = mount(TreeField, {
      attachTo: document.body,
      props: {
        value: 'archive.tar.gz',
        name: 'File name',
        isFolder: false,
      },
    })

    wrapper.vm.focus()
    const input = wrapper.find<HTMLInputElement>('input').element
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe(11) // 'archive.tar'
    wrapper.unmount()
  })

  it('selects the entire name for hidden dotfiles', () => {
    const wrapper = mount(TreeField, {
      attachTo: document.body,
      props: {
        value: '.gitignore',
        name: 'File name',
        isFolder: false,
      },
    })

    wrapper.vm.focus()
    const input = wrapper.find<HTMLInputElement>('input').element
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe(10) // '.gitignore'
    wrapper.unmount()
  })

  it('selects the entire name for extensionless files', () => {
    const wrapper = mount(TreeField, {
      attachTo: document.body,
      props: {
        value: 'Makefile',
        name: 'File name',
        isFolder: false,
      },
    })

    wrapper.vm.focus()
    const input = wrapper.find<HTMLInputElement>('input').element
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe(8) // 'Makefile'
    wrapper.unmount()
  })

  it('selects the entire name for folders even if they contain dots', () => {
    const wrapper = mount(TreeField, {
      attachTo: document.body,
      props: {
        value: 'release.v1.0',
        name: 'Folder name',
        isFolder: true,
      },
    })

    wrapper.vm.focus()
    const input = wrapper.find<HTMLInputElement>('input').element
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe(12) // 'release.v1.0'
    wrapper.unmount()
  })

  it('emits rename on Enter', async () => {
    const wrapper = mount(TreeField, {
      props: {
        value: 'old-name.ts',
        name: 'File name',
      },
    })

    await wrapper.find('input').setValue('new-name.ts')
    await wrapper.find('input').trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('rename')).toStrictEqual([['new-name.ts']])
  })

  it('emits abandon on Escape', async () => {
    const wrapper = mount(TreeField, {
      props: {
        value: 'old-name.ts',
        name: 'File name',
      },
    })

    await wrapper.find('input').setValue('new-name.ts')
    await wrapper.find('input').trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('abandon')).toHaveLength(1)
    expect(wrapper.emitted('rename')).toBeUndefined()
  })

  it('emits blur when input loses focus', async () => {
    const wrapper = mount(TreeField, {
      props: {
        value: 'old-name.ts',
        name: 'File name',
      },
    })

    await wrapper.find('input').trigger('blur')
    expect(wrapper.emitted('blur')).toHaveLength(1)
  })
})
