/**
 * 设置页 - 系统配置
 * 模型配置入口（跳转模型管理）、关于信息
 * 响应式布局
 */
import {
  Settings2,
  Cpu,
  Info,
  Flame,
  ExternalLink,
  Heart,
  ArrowRight,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import Layout from '@/components/Layout'

export default function Settings() {
  return (
    <Layout>
      {/* 页面头部 */}
      <div className="flex items-center gap-3 mb-6">
        <Settings2 className="w-6 h-6 text-gold-400" />
        <div>
          <h1 className="page-title">设置</h1>
          <p className="page-subtitle">配置系统参数</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 左侧：主要设置 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 模型管理入口 */}
          <section className="dao-card p-5 md:p-6">
            <div className="flex items-center gap-2 mb-4">
              <Cpu className="w-5 h-5 text-gold-400" />
              <h2 className="text-lg font-serif font-bold text-gold-300">语言模型</h2>
            </div>
            <p className="text-sm text-ink-400 mb-4 leading-relaxed">
              论道与丹性合成所用的语言模型（服务商、Base URL、API Key、温度等）已迁移至「模型管理」统一配置。
            </p>
            <Link to="/models" className="dao-btn-primary inline-flex">
              <Cpu className="w-4 h-4" />
              前往模型管理
              <ArrowRight className="w-4 h-4" />
            </Link>
          </section>
        </div>

        {/* 右侧：关于信息 */}
        <div className="space-y-6">
          <section className="dao-card p-5">
            <div className="flex items-center gap-2 mb-4">
              <Info className="w-5 h-5 text-gold-400" />
              <h2 className="text-base font-serif font-bold text-gold-300">关于炼丹炉</h2>
            </div>

            <div className="flex flex-col items-center text-center py-4">
              <div className="w-16 h-16 rounded-2xl bg-gradient-to-b from-cinnabar-500/20 to-ink-800/80 border-2 border-gold-500/30 flex items-center justify-center mb-3 glow-gold">
                <Flame className="w-8 h-8 text-cinnabar-400" />
              </div>
              <h3 className="text-lg font-serif font-bold text-gold-300 mb-1">炼丹炉</h3>
              <p className="text-xs text-ink-400 mb-4">v1.0.0</p>

              <p className="text-sm text-ink-400 leading-relaxed mb-4">
                以道教炼丹文化为设计灵感的金丹化性系统。
                金丹即语言模式技能包，道人即 AI Agent，服丹化性，围炉论道。
              </p>

              <div className="dao-divider text-[10px] w-full mb-4">
                <Heart className="w-3 h-3" />
              </div>

              <div className="space-y-2 w-full text-left">
                <a
                  href="https://github.com/yusanwen-code/alchemy-furnace"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 p-2.5 rounded-lg bg-ink-800/50 hover:bg-gold-400/5 border border-bronze-600/20 hover:border-gold-400/30 transition-all text-sm"
                >
                  <ExternalLink className="w-4 h-4 text-gold-400" />
                  <span className="text-rice-paper-200">GitHub 仓库</span>
                </a>
              </div>
            </div>
          </section>

          {/* 技术栈 */}
          <section className="dao-card p-5">
            <h3 className="text-sm font-medium text-gold-300 mb-3">技术栈</h3>
            <div className="flex flex-wrap gap-2">
              {['React 18', 'TypeScript', 'Tailwind CSS', 'Vite', 'Go API', 'Python 语言引擎', 'PostgreSQL'].map(tech => (
                <span
                  key={tech}
                  className="px-2.5 py-1 text-[11px] rounded-full bg-jade-500/10 text-jade-400 border border-jade-500/20"
                >
                  {tech}
                </span>
              ))}
            </div>
          </section>
        </div>
      </div>
    </Layout>
  )
}
