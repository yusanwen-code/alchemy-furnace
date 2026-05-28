/**
 * 拖拽上传组件 - 道教符文装饰风格
 * 支持拖拽上传、多文件选择、上传进度显示
 */
import { useCallback, useState, useRef } from 'react'
import { Upload, File, X, Check, Loader2 } from 'lucide-react'

/** 上传文件状态 */
interface UploadFile {
  id: string
  file: File
  name: string
  size: number
  progress: number
  status: 'pending' | 'uploading' | 'done' | 'error'
}

interface UploadDropzoneProps {
  /** 上传回调 */
  onUpload: (files: FileList) => void
  /** 是否正在上传 */
  uploading?: boolean
  /** 接受的文件类型 */
  accept?: string
}

export default function UploadDropzone({ onUpload, uploading = false, accept }: UploadDropzoneProps) {
  const [isDragOver, setIsDragOver] = useState(false)
  const [selectedFiles, setSelectedFiles] = useState<UploadFile[]>([])
  const inputRef = useRef<HTMLInputElement>(null)

  /** 处理拖拽进入 */
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(true)
  }, [])

  /** 处理拖拽离开 */
  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)
  }, [])

  /** 处理文件放下 */
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)

    const files = e.dataTransfer.files
    if (files.length > 0) {
      handleFiles(files)
    }
  }, [])

  /** 处理文件选择 */
  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (files && files.length > 0) {
      handleFiles(files)
    }
  }, [])

  /** 处理文件列表 */
  const handleFiles = (files: FileList) => {
    const newFiles: UploadFile[] = Array.from(files).map(file => ({
      id: Math.random().toString(36).substring(7),
      file,
      name: file.name,
      size: file.size,
      progress: 0,
      status: 'pending' as const,
    }))
    setSelectedFiles(prev => [...prev, ...newFiles])
  }

  /** 移除已选文件 */
  const removeFile = (id: string) => {
    setSelectedFiles(prev => prev.filter(f => f.id !== id))
  }

  /** 执行上传 */
  const executeUpload = () => {
    if (selectedFiles.length === 0) return

    const dt = new DataTransfer()
    selectedFiles.forEach(f => dt.items.add(f.file))
    onUpload(dt.files)
    setSelectedFiles([])
  }

  /** 格式化文件大小 */
  const formatSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  return (
    <div className="space-y-4">
      {/* 拖拽区域 */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
        className={`
          relative border-2 border-dashed rounded-xl p-6 md:p-8
          flex flex-col items-center justify-center gap-3
          cursor-pointer transition-all duration-300
          ${isDragOver
            ? 'border-gold-400 bg-gold-400/10 scale-[1.02]'
            : 'border-bronze-600/30 bg-ink-800/40 hover:border-bronze-500/50 hover:bg-ink-800/60'
          }
        `}
      >
        {/* 隐藏的文件输入 */}
        <input
          ref={inputRef}
          type="file"
          multiple
          accept={accept}
          onChange={handleFileSelect}
          className="hidden"
        />

        {/* 上传图标 */}
        <div className={`
          w-14 h-14 rounded-full flex items-center justify-center
          ${isDragOver ? 'bg-gold-500/20 text-gold-400' : 'bg-bronze-600/10 text-bronze-400'}
          transition-colors duration-300
        `}>
          <Upload className="w-7 h-7" />
        </div>

        {/* 提示文字 */}
        <div className="text-center">
          <p className="text-sm text-rice-paper-200">
            {isDragOver ? '松开以上传丹方' : '点击或拖拽上传丹方'}
          </p>
          <p className="text-xs text-ink-400 mt-1">
            支持 Word、Excel、PDF、Markdown、文本、音频、视频等格式
          </p>
        </div>

        {/* 道教符文装饰 */}
        <div className="absolute top-2 left-2 text-gold-500/10 text-lg font-serif select-none">☰</div>
        <div className="absolute top-2 right-2 text-gold-500/10 text-lg font-serif select-none">☷</div>
        <div className="absolute bottom-2 left-2 text-gold-500/10 text-lg font-serif select-none">☵</div>
        <div className="absolute bottom-2 right-2 text-gold-500/10 text-lg font-serif select-none">☲</div>
      </div>

      {/* 已选文件列表 */}
      {selectedFiles.length > 0 && (
        <div className="space-y-2 animate-fade-in">
          {selectedFiles.map(file => (
            <div
              key={file.id}
              className="flex items-center gap-3 p-3 rounded-lg bg-ink-800/60 border border-bronze-600/20"
            >
              <File className="w-5 h-5 text-gold-400 flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-sm text-rice-paper-100 truncate">{file.name}</p>
                <p className="text-xs text-ink-400">{formatSize(file.size)}</p>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  removeFile(file.id)
                }}
                className="p-1 rounded hover:bg-cinnabar-500/20 text-ink-400 hover:text-cinnabar-400 transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          ))}

          {/* 上传按钮 */}
          <button
            onClick={executeUpload}
            disabled={uploading}
            className="dao-btn-primary w-full mt-2"
          >
            {uploading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                上传中...
              </>
            ) : (
              <>
                <Check className="w-4 h-4" />
                确认上传 ({selectedFiles.length} 个文件)
              </>
            )}
          </button>
        </div>
      )}
    </div>
  )
}
