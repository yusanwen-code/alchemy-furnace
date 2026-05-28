/**
 * 金丹详情页面 - 丹方管理
 * 金丹信息头部 + 丹方列表 + 上传功能
 * H5 优化: 卡片式列表替代表格
 */
import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  CircleDot,
  FileText,
  Upload,
  Trash2,
  RefreshCw,
  Loader2,
  FlaskConical,
  Clock,
  AlertCircle,
  File,
} from 'lucide-react'
import { usePill } from '@/contexts/PillContext'
import UploadDropzone from '@/components/UploadDropzone'
import Layout from '@/components/Layout'
import { formatFileSize, getFileTypeInfo, formatDateTime, EXTRACT_STATUS_MAP } from '@/utils/format'
import type { Recipe } from '@/services/types'

export default function PillDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const pillId = Number(id)

  const { state, fetchPill, fetchRecipes, uploadRecipes, removeRecipe, reExtractRecipe } = usePill()
  const [showUpload, setShowUpload] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [reExtractingId, setReExtractingId] = useState<number | null>(null)

  const pill = state.currentPill
  const recipes = state.currentRecipes

  // 加载数据
  useEffect(() => {
    if (pillId) {
      fetchPill(pillId)
      fetchRecipes(pillId)
    }
  }, [pillId, fetchPill, fetchRecipes])

  /** 处理上传 */
  const handleUpload = async (files: FileList) => {
    setUploading(true)
    await uploadRecipes(pillId, files)
    setUploading(false)
    setShowUpload(false)
  }

  /** 处理重新提取 */
  const handleReExtract = async (recipeId: number) => {
    setReExtractingId(recipeId)
    await reExtractRecipe(recipeId)
    // 刷新列表
    setTimeout(async () => {
      await fetchRecipes(pillId)
      setReExtractingId(null)
    }, 2500)
  }

  /** 获取文件类型图标颜色 */
  const getFileIconColor = (filename: string): string => {
    const info = getFileTypeInfo(filename)
    return info.color
  }

  if (!pill && state.loading) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center py-16">
          <Loader2 className="w-8 h-8 text-gold-400 animate-spin mb-3" />
          <p className="text-sm text-ink-400">加载中...</p>
        </div>
      </Layout>
    )
  }

  if (!pill) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center py-16">
          <AlertCircle className="w-12 h-12 text-cinnabar-400 mb-3" />
          <p className="text-sm text-ink-400">金丹不存在</p>
          <Link to="/pills" className="dao-btn-primary mt-4">
            <ArrowLeft className="w-4 h-4" />
            返回金丹阁
          </Link>
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      {/* 返回按钮 */}
      <Link
        to="/pills"
        className="inline-flex items-center gap-1.5 text-sm text-ink-400 hover:text-gold-300 transition-colors mb-4"
      >
        <ArrowLeft className="w-4 h-4" />
        返回金丹阁
      </Link>

      {/* 金丹信息头部 */}
      <div className="dao-card p-5 md:p-6 mb-6">
        <div className="flex flex-col md:flex-row md:items-start gap-4">
          {/* 图标 */}
          <div className={`
            flex-shrink-0 w-16 h-16 rounded-2xl flex items-center justify-center
            ${pill.status === 'refined'
              ? 'bg-gold-500/15 text-gold-400 glow-gold'
              : pill.status === 'refining'
                ? 'bg-jade-500/15 text-jade-400'
                : 'bg-cinnabar-500/15 text-cinnabar-400'
            }
          `}>
            <FlaskConical className="w-8 h-8" />
          </div>

          {/* 信息 */}
          <div className="flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-2 mb-2">
              <h1 className="text-xl md:text-2xl font-serif font-bold text-rice-paper-100">
                {pill.name}
              </h1>
              <span className={EXTRACT_STATUS_MAP[pill.status]?.badgeClass || ''}>
                {pill.status === 'refined' ? '已成丹' : pill.status === 'refining' ? '炼制中' : '失败'}
              </span>
            </div>

            {pill.description && (
              <p className="text-sm text-ink-400 mb-3">{pill.description}</p>
            )}

            <div className="flex flex-wrap items-center gap-4 text-xs text-ink-400">
              <span className="flex items-center gap-1">
                <FileText className="w-3.5 h-3.5" />
                {recipes.length} 个丹方
              </span>
              <span className="flex items-center gap-1">
                <CircleDot className="w-3.5 h-3.5" />
                {pill.vector_count} 个向量
              </span>
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                {formatDateTime(pill.created_at)}
              </span>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowUpload(!showUpload)}
              className="dao-btn-primary"
            >
              <Upload className="w-4 h-4" />
              上传丹方
            </button>
          </div>
        </div>
      </div>

      {/* 上传区域 */}
      {showUpload && (
        <div className="dao-card p-5 mb-6 animate-fade-in">
          <h3 className="text-sm font-medium text-gold-300 mb-3">上传新丹方</h3>
          <UploadDropzone onUpload={handleUpload} uploading={uploading} />
        </div>
      )}

      {/* 丹方列表 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-serif font-bold text-gold-300 flex items-center gap-2">
            <FileText className="w-5 h-5" />
            丹方列表
          </h2>
          <span className="text-xs text-ink-400">共 {recipes.length} 个丹方</span>
        </div>

        {/* 桌面端表格 */}
        <div className="hidden md:block dao-card overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-ink-800/80 border-b border-bronze-600/20">
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">文件名</th>
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">类型</th>
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">大小</th>
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">提取状态</th>
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">Chunk 数</th>
                <th className="text-left px-4 py-3 text-gold-300/80 font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {recipes.map((recipe: Recipe) => {
                const extractInfo = EXTRACT_STATUS_MAP[recipe.extract_status]
                return (
                  <tr
                    key={recipe.id}
                    className="border-b border-ink-700/30 hover:bg-gold-400/5 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <File className={`w-4 h-4 ${getFileIconColor(recipe.filename)}`} />
                        <span className="text-rice-paper-100">{recipe.filename}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-ink-400 uppercase">{recipe.file_type}</td>
                    <td className="px-4 py-3 text-ink-400">{formatFileSize(recipe.file_size)}</td>
                    <td className="px-4 py-3">
                      <span className={extractInfo?.badgeClass || ''}>
                        {extractInfo?.label || recipe.extract_status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-ink-400">
                      {recipe.chunk_count > 0 ? recipe.chunk_count : '-'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => handleReExtract(recipe.id)}
                          disabled={reExtractingId === recipe.id}
                          className="p-1.5 rounded hover:bg-jade-500/20 text-ink-400 hover:text-jade-400 transition-colors"
                          title="重新提取"
                        >
                          {reExtractingId === recipe.id ? (
                            <Loader2 className="w-4 h-4 animate-spin" />
                          ) : (
                            <RefreshCw className="w-4 h-4" />
                          )}
                        </button>
                        <button
                          onClick={() => removeRecipe(recipe.id)}
                          className="p-1.5 rounded hover:bg-cinnabar-500/20 text-ink-400 hover:text-cinnabar-400 transition-colors"
                          title="删除"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>

        {/* H5 卡片式列表 */}
        <div className="md:hidden space-y-3">
          {recipes.map((recipe: Recipe) => {
            const extractInfo = EXTRACT_STATUS_MAP[recipe.extract_status]
            return (
              <div
                key={recipe.id}
                className="dao-card p-4 flex items-start gap-3"
              >
                <File className={`w-8 h-8 flex-shrink-0 ${getFileIconColor(recipe.filename)}`} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-rice-paper-100 truncate">{recipe.filename}</p>
                  <div className="flex items-center gap-2 mt-1 text-xs text-ink-400">
                    <span className="uppercase">{recipe.file_type}</span>
                    <span>·</span>
                    <span>{formatFileSize(recipe.file_size)}</span>
                  </div>
                  <div className="flex items-center justify-between mt-2">
                    <span className={extractInfo?.badgeClass || ''}>
                      {extractInfo?.label || recipe.extract_status}
                    </span>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleReExtract(recipe.id)}
                        disabled={reExtractingId === recipe.id}
                        className="p-1.5 rounded hover:bg-jade-500/20 text-ink-400 hover:text-jade-400 transition-colors"
                      >
                        {reExtractingId === recipe.id ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <RefreshCw className="w-3.5 h-3.5" />
                        )}
                      </button>
                      <button
                        onClick={() => removeRecipe(recipe.id)}
                        className="p-1.5 rounded hover:bg-cinnabar-500/20 text-ink-400 hover:text-cinnabar-400 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>

        {/* 空状态 */}
        {recipes.length === 0 && !state.loading && (
          <div className="dao-card flex flex-col items-center py-12 text-center">
            <FileText className="w-12 h-12 text-ink-600 mb-3" />
            <p className="text-sm text-ink-400 mb-1">暂无丹方</p>
            <p className="text-xs text-ink-500 mb-4">上传文档文件以丰富这颗金丹</p>
            <button onClick={() => setShowUpload(true)} className="dao-btn-primary">
              <Upload className="w-4 h-4" />
              上传丹方
            </button>
          </div>
        )}
      </div>
    </Layout>
  )
}
