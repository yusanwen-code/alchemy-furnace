import next from 'eslint-config-next'

export default [
  ...next,
  {
    ignores: ['node_modules/', '.next/', 'out/', 'dist/', 'legacy/'],
  },
]
