/**
 * 应用入口文件
 * 挂载 React 应用到 DOM
 */
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles/globals.css'

/**
 * 创建 React 根节点并渲染应用
 * 使用 StrictMode 进行开发时额外的检查
 */
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
