import path from 'path'
import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

const shared = path.resolve(__dirname, '../../../frontend-shared')

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      '@shared': shared,
    },
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname), shared],
    },
  },
})
