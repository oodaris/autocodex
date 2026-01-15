/* eslint-disable react-hooks/exhaustive-deps -- useAsync accepts deps from caller */
import { useEffect, useRef } from 'react'
import type { DependencyList } from 'react'

type AsyncEffect = (signal: AbortSignal) => Promise<void> | void

export function useAsync(effect: AsyncEffect, deps: DependencyList): void {
  const effectRef = useRef(effect)

  useEffect(() => {
    effectRef.current = effect
  }, [effect])

  useEffect(() => {
    const controller = new AbortController()
    void effectRef.current(controller.signal)
    return () => {
      controller.abort()
    }
  }, deps)
}
