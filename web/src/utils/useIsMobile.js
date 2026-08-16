import { useEffect, useState } from 'react'

// Track the mobile breakpoint (< 768px) with an accurate initial value to
// avoid layout flash on first paint.
export function useIsMobile() {
  const [isMobile, setIsMobile] = useState(() => {
    return typeof window !== 'undefined' && window.innerWidth < 768
  })

  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < 768)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  return isMobile
}
