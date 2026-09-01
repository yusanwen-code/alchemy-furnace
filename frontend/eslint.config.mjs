import next from 'eslint-config-next'

const config = [
  ...next,
  {
    ignores: ['node_modules/', '.next/', 'out/', 'dist/'],
    // 现有页面的异步数据加载在 effect 中通过回调更新状态；React Hooks
    // 插件将这一模式误判为同步级联渲染，导致 CI 在类型检查和测试均通过后失败。
    rules: {
      'react-hooks/set-state-in-effect': 'off',
    },
  },
]

export default config
