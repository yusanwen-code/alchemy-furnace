import { get } from './api'

export interface DesktopDiagnostics {
  timestamp: number
  log_dir: string
  app_log: string
  python_log: string
  python_engine: 'ok' | 'down' | 'unknown'
}

export function getDiagnostics(): Promise<DesktopDiagnostics> {
  return get<DesktopDiagnostics>('/system/diagnostics')
}
