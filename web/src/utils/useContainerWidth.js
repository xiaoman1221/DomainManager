import { useEffect, useRef, useState } from 'react'

// Tracks an element's rendered width with ResizeObserver so columns/layout
// can adapt dynamically to the available space (desktop resize, mobile, etc).
export function useContainerWidth() {
  const ref = useRef(null)
  const [width, setWidth] = useState(0)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const update = () => setWidth(el.clientWidth)
    update()
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  return [ref, width]
}
