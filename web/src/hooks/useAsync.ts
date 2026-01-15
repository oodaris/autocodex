/* eslint-disable react-hooks/exhaustive-deps -- useAsync accepts deps from caller */
import { useEffect } from 'react'
import type { DependencyList } from 'react'

type AsyncEffect = (signal: AbortSignal) => Promise<void> | void

export function useAsync(effect: AsyncEffect, deps: DependencyList): void {
  useEffect(() => {
    const controller = new AbortController()
    void effect(controller.signal)
    return () => {
      controller.abort()
    }
  }, [effect, ...deps])
}
