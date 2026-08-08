'use client'

/**
 * Markdown 渲染组件
 * 支持代码高亮、表格、列表等常见 Markdown 语法
 * 使用 react-markdown + remark-gfm + highlight.js（浅色主题）
 */
import { useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkBreaks from 'remark-breaks'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.min.css'

interface MarkdownRendererProps {
  content: string
}

export function MarkdownRenderer({ content }: MarkdownRendererProps) {
  // 代码高亮
  useEffect(() => {
    hljs.highlightAll()
  }, [content])

  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={{
          // 代码块
          code({ className, children, ...props }) {
            const match = /language-(\w+)/.exec(className || '')
            const language = match ? match[1] : ''
            const isInline = !className

            if (isInline) {
              return (
                <code className="bg-muted text-sage px-1.5 py-0.5 rounded text-xs" {...props}>
                  {children}
                </code>
              )
            }

            return (
              <div className="my-2 rounded-lg overflow-hidden border border-border/70">
                {/* 代码块头部 */}
                <div className="flex items-center justify-between px-3 py-1.5 bg-paper-deep border-b border-border/70">
                  <span className="text-[10px] text-muted-foreground uppercase">
                    {language || 'text'}
                  </span>
                </div>
                <pre className="m-0 p-0">
                  <code className={`hljs ${language}`}>
                    {String(children).replace(/\n$/, '')}
                  </code>
                </pre>
              </div>
            )
          },
          // 引用块
          blockquote({ children }) {
            return (
              <blockquote className="border-l-2 border-gold/50 bg-gold/5 pl-4 py-1 pr-2 my-2 rounded-r">
                {children}
              </blockquote>
            )
          },
          // 表格
          table({ children }) {
            return (
              <div className="overflow-x-auto my-2">
                <table className="w-full text-xs border-collapse">
                  {children}
                </table>
              </div>
            )
          },
          thead({ children }) {
            return <thead className="bg-muted">{children}</thead>
          },
          th({ children }) {
            return (
              <th className="text-left px-3 py-2 text-gold font-medium border border-border/70">
                {children}
              </th>
            )
          },
          td({ children }) {
            return (
              <td className="px-3 py-2 text-foreground/80 border border-border/70">
                {children}
              </td>
            )
          },
          // 列表
          ul({ children }) {
            return <ul className="list-disc list-inside space-y-0.5 my-1">{children}</ul>
          },
          ol({ children }) {
            return <ol className="list-decimal list-inside space-y-0.5 my-1">{children}</ol>
          },
          // 链接
          a({ children, href }) {
            return (
              <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                {children}
              </a>
            )
          },
          // 段落
          p({ children }) {
            return <p className="my-1.5 leading-relaxed">{children}</p>
          },
          // 标题
          h1({ children }) {
            return <h1 className="text-lg font-bold text-foreground mt-4 mb-2 font-serif">{children}</h1>
          },
          h2({ children }) {
            return <h2 className="text-base font-bold text-foreground mt-3 mb-2 font-serif">{children}</h2>
          },
          h3({ children }) {
            return <h3 className="text-sm font-bold text-foreground mt-2 mb-1 font-serif">{children}</h3>
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
