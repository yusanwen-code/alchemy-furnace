export interface RecentApiFailure {
  at: string
  method: string
  path: string
  status: number
  errorCode?: string
  requestId?: string
  category: 'business' | 'gateway' | 'engine' | 'upstream' | 'unknown'
}

const MAX_FAILURES = 20
let failures: RecentApiFailure[] = []

export function recordApiFailure(failure: RecentApiFailure): void {
  failures = [failure, ...failures].slice(0, MAX_FAILURES)
}

export function listApiFailures(): RecentApiFailure[] {
  return failures.map(failure => ({ ...failure }))
}

export function clearApiFailures(): void {
  failures = []
}
