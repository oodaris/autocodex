import { useEffect, useRef, useState } from 'react'

type PollingOptions = {
  enabled: boolean
  intervalMs: number
  maxIntervalMs?: number
  backoffFactor?: number
  pauseWhenHidden?: boolean
}

type PollTask = (signal: AbortSignal) => Promise<void>

export function usePolling(task: PollTask, options: PollingOptions): number {
  const {
    enabled,
    intervalMs,
    maxIntervalMs = intervalMs * 4,
    backoffFactor = 1.6,
    pauseWhenHidden = true,
  } = options
  const [currentDelay, setCurrentDelay] = useState(intervalMs)
  const [isVisible, setIsVisible] = useState(() => {
    if (typeof document === 'undefined') return true
    return document.visibilityState !== 'hidden'
  })
  const delayRef = useRef(intervalMs)
  const timerRef = useRef<number | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const runningRef = useRef(false)

  useEffect(() => {
    if (!pauseWhenHidden || typeof document === 'undefined') return
    const handleVisibility = () => {
      setIsVisible(document.visibilityState !== 'hidden')
    }
    document.addEventListener('visibilitychange', handleVisibility)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibility)
    }
  }, [pauseWhenHidden])

  useEffect(() => {
    delayRef.current = intervalMs
    setCurrentDelay(intervalMs)
  }, [intervalMs])

  useEffect(() => {
    const shouldPoll = enabled && (!pauseWhenHidden || isVisible)

    if (!shouldPoll) {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current)
        timerRef.current = null
      }
      abortRef.current?.abort()
      abortRef.current = null
      delayRef.current = intervalMs
      setCurrentDelay(intervalMs)
      return
    }

    let active = true

    const schedule = () => {
      if (!active) return
      timerRef.current = window.setTimeout(() => {
        void tick()
      }, delayRef.current)
    }

    const tick = async () => {
      if (!active || runningRef.current) {
        schedule()
        return
      }
      runningRef.current = true
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller
      try {
        await task(controller.signal)
        if (!active) return
        delayRef.current = intervalMs
        setCurrentDelay(delayRef.current)
      } catch (err) {
        if (!active) return
        if (!(err instanceof DOMException && err.name === 'AbortError')) {
          delayRef.current = Math.min(maxIntervalMs, Math.ceil(delayRef.current * backoffFactor))
          setCurrentDelay(delayRef.current)
        }
      } finally {
        runningRef.current = false
        schedule()
      }
    }

    schedule()

    return () => {
      active = false
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current)
        timerRef.current = null
      }
      abortRef.current?.abort()
      abortRef.current = null
    }
  }, [backoffFactor, enabled, intervalMs, isVisible, maxIntervalMs, pauseWhenHidden, task])

  return currentDelay
}
