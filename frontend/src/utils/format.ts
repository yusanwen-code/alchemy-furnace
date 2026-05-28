/**
 * 格式化工具函数
 * 提供文件大小、时间、文件类型等格式化功能
 */

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的字符串 (如: "1.5 MB")
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + units[i]
}

/**
 * 格式化日期时间
 * @param date 日期字符串或 Date 对象
 * @returns 格式化后的字符串 (如: "2024年1月1日 12:00")
 */
export function formatDateTime(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const now = new Date()
  const diff = now.getTime() - d.getTime()

  // 小于 1 分钟
  if (diff < 60 * 1000) return '刚刚'
  // 小于 1 小时
  if (diff < 60 * 60 * 1000) return `${Math.floor(diff / (60 * 1000))} 分钟前`
  // 小于 24 小时
  if (diff < 24 * 60 * 60 * 1000) return `${Math.floor(diff / (60 * 60 * 1000))} 小时前`
  // 小于 7 天
  if (diff < 7 * 24 * 60 * 60 * 1000) return `${Math.floor(diff / (24 * 60 * 60 * 1000))} 天前`

  // 默认返回完整日期
  const year = d.getFullYear()
  const month = d.getMonth() + 1
  const day = d.getDate()
  const hour = d.getHours().toString().padStart(2, '0')
  const minute = d.getMinutes().toString().padStart(2, '0')
  return `${year}年${month}月${day}日 ${hour}:${minute}`
}

/**
 * 格式化日期（仅日期部分）
 * @param date 日期字符串或 Date 对象
 * @returns 格式化后的字符串 (如: "2024年1月1日")
 */
export function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const year = d.getFullYear()
  const month = d.getMonth() + 1
  const day = d.getDate()
  return `${year}年${month}月${day}日`
}

/**
 * 文件类型到图标和颜色的映射
 */
export const FILE_TYPE_MAP: Record<string, { icon: string; color: string; label: string }> = {
  doc: { icon: 'FileText', color: 'text-blue-400', label: 'Word' },
  docx: { icon: 'FileText', color: 'text-blue-400', label: 'Word' },
  xls: { icon: 'Table', color: 'text-green-400', label: 'Excel' },
  xlsx: { icon: 'Table', color: 'text-green-400', label: 'Excel' },
  md: { icon: 'BookOpen', color: 'text-purple-400', label: 'Markdown' },
  txt: { icon: 'FileText', color: 'text-gray-400', label: '文本' },
  pdf: { icon: 'FileText', color: 'text-red-400', label: 'PDF' },
  mp3: { icon: 'Music', color: 'text-yellow-400', label: '音频' },
  wav: { icon: 'Music', color: 'text-yellow-400', label: '音频' },
  m4a: { icon: 'Music', color: 'text-yellow-400', label: '音频' },
  mp4: { icon: 'Video', color: 'text-pink-400', label: '视频' },
  avi: { icon: 'Video', color: 'text-pink-400', label: '视频' },
  mov: { icon: 'Video', color: 'text-pink-400', label: '视频' },
}

/**
 * 获取文件类型信息
 * @param filename 文件名
 * @returns 文件类型信息
 */
export function getFileTypeInfo(filename: string): { icon: string; color: string; label: string } {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  return FILE_TYPE_MAP[ext] || { icon: 'File', color: 'text-gray-400', label: ext.toUpperCase() }
}

/**
 * 提取状态映射
 */
export const EXTRACT_STATUS_MAP: Record<string, { label: string; badgeClass: string }> = {
  pending: { label: '待提取', badgeClass: 'dao-badge-pending' },
  extracting: { label: '提取中', badgeClass: 'dao-badge-refining' },
  completed: { label: '已完成', badgeClass: 'dao-badge-refined' },
  failed: { label: '失败', badgeClass: 'dao-badge-failed' },
}

/**
 * 金丹状态映射
 */
export const PILL_STATUS_MAP: Record<string, { label: string; badgeClass: string }> = {
  refining: { label: '炼制中', badgeClass: 'dao-badge-refining' },
  refined: { label: '已成丹', badgeClass: 'dao-badge-refined' },
  failed: { label: '炼制失败', badgeClass: 'dao-badge-failed' },
}

/**
 * 截断文本
 * @param text 原文本
 * @param maxLength 最大长度
 * @returns 截断后的文本
 */
export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}
