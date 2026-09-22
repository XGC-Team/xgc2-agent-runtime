import { defineConfig } from 'vitest/config'

export default defineConfig({
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'jsdom',
    // base-ui popup transitions await requestAnimationFrame; jsdom only fires it
    // with pretendToBeVisual, otherwise popup teardown stalls for minutes.
    environmentOptions: { jsdom: { pretendToBeVisual: true } },
    include: ['test/**/*.test.{ts,tsx}', 'src/upstream/t3/**/*.test.tsx'],
    restoreMocks: true,
  },
})
