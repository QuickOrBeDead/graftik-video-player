export type LogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error'

export interface LogEntry {
  id: number
  time: Date
  level: LogLevel
  message: string
  source?: string
  attrs?: Record<string, unknown>
  fromBackend?: boolean
}

type LogCallback = (entry: LogEntry) => void

const LEVEL_PRIORITY: Record<LogLevel, number> = {
  trace: 0,
  debug: 1,
  info: 2,
  warn: 3,
  error: 4,
}

class FrontendLogger {
  private level: LogLevel = 'debug'
  private buffer: LogEntry[] = []
  private _maxBuffer = 1000
  get maxBuffer(): number { return this._maxBuffer }
  set maxBuffer(value: number) { this._maxBuffer = Math.max(1, value) }
  private nextId = 0
  private listeners: Set<LogCallback> = new Set()
  private prefix = '[Graftik]'

  setLevel(level: LogLevel) {
    this.level = level
  }

  getLevel(): LogLevel {
    return this.level
  }

  subscribe(cb: LogCallback): () => void {
    this.listeners.add(cb)
    return () => this.listeners.delete(cb)
  }

  getBuffer(): LogEntry[] {
    return this.buffer
  }

  clear() {
    this.buffer = []
    this.notifyListeners()
  }

  trace(...args: unknown[]) {
    this.log('trace', args, false)
  }

  debug(...args: unknown[]) {
    this.log('debug', args, false)
  }

  info(...args: unknown[]) {
    this.log('info', args, false)
  }

  warn(...args: unknown[]) {
    this.log('warn', args, false)
  }

  error(...args: unknown[]) {
    this.log('error', args, false)
  }

  fromBackend(entry: { level: string; message: string; time?: string; source?: string; attrs?: Record<string, unknown> }) {
    const level = (entry.level?.toLowerCase() as LogLevel) || 'info'
    if (!this.shouldLog(level)) {
      return
    }

    const logEntry: LogEntry = {
      id: this.nextId++,
      time: entry.time ? new Date(entry.time) : new Date(),
      level,
      message: entry.message,
      source: entry.source,
      attrs: entry.attrs,
      fromBackend: true,
    }

    this.buffer.push(logEntry)
    if (this.buffer.length > this._maxBuffer) {
      this.buffer.shift()
    }

    this.notifyListeners()
  }

  private log(level: LogLevel, args: unknown[], fromBackend: boolean) {
    if (!this.shouldLog(level)) return

    const message = args.map(a => (typeof a === 'object' ? this.safeStringify(a) : String(a))).join(' ')
    const logEntry: LogEntry = {
      id: this.nextId++,
      time: new Date(),
      level,
      message,
      fromBackend,
    }

    this.buffer.push(logEntry)
    if (this.buffer.length > this._maxBuffer) {
      this.buffer.shift()
    }

    const consoleFn = console[level] || console.log
    consoleFn(this.prefix, ...args)

    this.notifyListeners()
  }

  private shouldLog(level: LogLevel): boolean {
    return LEVEL_PRIORITY[level] >= LEVEL_PRIORITY[this.level]
  }

  private safeStringify(obj: unknown): string {
    try {
      return JSON.stringify(obj)
    } catch {
      return String(obj)
    }
  }

  private notifyListeners() {
    const latest = this.buffer[this.buffer.length - 1]
    if (latest) {
      this.listeners.forEach(cb => cb(latest))
    }
  }
}

export const logger = new FrontendLogger()
