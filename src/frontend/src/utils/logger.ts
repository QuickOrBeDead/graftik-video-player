import { Log as sendToBackend } from '../../wailsjs/go/main/App'

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
  private _paused = false
  get paused(): boolean { return this._paused }
  set paused(value: boolean) { this._paused = value }
  private nextId = 0
  private listeners: Set<LogCallback> = new Set()
  private prefix = '[Graftik]'
  private backendReady = false
  private sendBuffer: Array<{ level: LogLevel; msg: string; attrs: Record<string, unknown> }> = []

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

  trace(msg: string, ...args: unknown[]) {
    this.log('trace', msg, args, false)
  }

  debug(msg: string, ...args: unknown[]) {
    this.log('debug', msg, args, false)
  }

  info(msg: string, ...args: unknown[]) {
    this.log('info', msg, args, false)
  }

  warn(msg: string, ...args: unknown[]) {
    this.log('warn', msg, args, false)
  }

  error(msg: string, ...args: unknown[]) {
    this.log('error', msg, args, false)
  }

  fromBackend(entry: { level: string; message: string; time?: string; source?: string; attrs?: Record<string, unknown> }) {
    if (this._paused) return
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

  flushBuffer() {
    this.backendReady = true
    const pending = this.sendBuffer.splice(0)
    for (const entry of pending) {
      this.sendToBackend(entry.level, entry.msg, entry.attrs)
    }
  }

  private log(level: LogLevel, msg: string, args: unknown[], fromBackend: boolean) {
    if (this._paused) return
    if (!this.shouldLog(level)) return

    const argsStr = args.map(a => (typeof a === 'object' ? this.safeStringify(a) : String(a))).join(' ')
    const message = argsStr ? `${msg} ${argsStr}` : msg
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
    consoleFn(this.prefix, msg, ...args)

    if (!fromBackend) {
      const attrs: Record<string, unknown> = {}
      for (let i = 0; i < args.length; i += 2) {
        attrs[String(args[i])] = i + 1 < args.length ? args[i + 1] : true
      }
      this.sendToBackend(level, msg, attrs)
    }

    this.notifyListeners()
  }

  private sendToBackend(level: LogLevel, msg: string, attrs?: Record<string, unknown>) {
    if (!this.backendReady) {
      this.sendBuffer.push({ level, msg, attrs: attrs || {} })
      return
    }

    sendToBackend(level, msg, attrs || {})
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
