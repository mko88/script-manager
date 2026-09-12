import path from 'path'
import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

const shared = path.resolve(__dirname, '../../../frontend-shared')

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    // frontend-shared sits outside this app and has no node_modules of its
    // own, so a bare import from a shared component has nowhere to resolve
    // from. Point those at this app's copy.
    alias: [
      {find: '@shared', replacement: shared},
      {
        find: /^(codemirror|@codemirror\/.*|@lezer\/.*)$/,
        replacement: path.resolve(__dirname, 'node_modules/$1'),
      },
    ],
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname), shared],
    },
  },
})
