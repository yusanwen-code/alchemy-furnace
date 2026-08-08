/**
 * 页面布局容器组件
 * 提供统一的页面结构：导航栏 + 内容区 + 底部栏
 * 响应式处理：H5 适配底部安全区域
 * 背景纹理：宣纸质感
 */
import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import Navbar from './Navbar'
import Footer from './Footer'

interface LayoutProps {
  children: React.ReactNode
  /** 是否显示底部栏（默认 true） */
  showFooter?: boolean
  /** 是否使用卷轴容器样式 */
  scrollStyle?: boolean
}

export default function Layout({ children, showFooter = true, scrollStyle = false }: LayoutProps) {
  const location = useLocation()

  /** 页面切换时滚动到顶部 */
  useEffect(() => {
    window.scrollTo(0, 0)
  }, [location.pathname])

  return (
    <div className="min-h-screen flex flex-col bg-ink-700 bg-rice-paper">
      {/* 顶部导航栏 */}
      <Navbar />

      {/* 主内容区 */}
      <main className={`
        flex-1 pt-16 md:pt-20 pb-20 md:pb-8
        px-4 sm:px-6 lg:px-8
        ${scrollStyle ? 'max-w-5xl' : 'max-w-7xl'}
        mx-auto w-full
      `}>
        <div className="animate-fade-in">
          {children}
        </div>
      </main>

      {/* 底部栏 */}
      {showFooter && <Footer />}
    </div>
  )
}
