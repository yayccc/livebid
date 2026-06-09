import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { BottomNav } from './BottomNav'

export function AppShell() {
  const location = useLocation()
  const [isKeyboardOpen, setIsKeyboardOpen] = useState(false)
  const isHome = location.pathname === '/'
  const isLightPage = location.pathname === '/messages' || location.pathname.startsWith('/profile')
  const shellClassName = [
    'app-shell',
    isHome ? 'app-shell--immersive' : '',
    isLightPage ? 'app-shell--light' : '',
    isKeyboardOpen ? 'app-shell--keyboard-open' : '',
  ]
    .filter(Boolean)
    .join(' ')

  useEffect(() => {
    let frame = 0
    let viewportBaseline = getVisualViewportHeight()

    function updateKeyboardState() {
      window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(() => {
        const isInputFocused = isTextInputElement(document.activeElement)
        const currentViewportHeight = getVisualViewportHeight()

        if (!isInputFocused) {
          viewportBaseline = currentViewportHeight
          setIsKeyboardOpen(false)
          return
        }

        const keyboardInset = Math.max(
          viewportBaseline - currentViewportHeight,
          window.innerHeight - currentViewportHeight - getVisualViewportOffsetTop(),
          0,
        )
        const keyboardVisible = keyboardInset > 120

        if (!keyboardVisible && currentViewportHeight > viewportBaseline) {
          viewportBaseline = currentViewportHeight
        }

        setIsKeyboardOpen(keyboardVisible)
      })
    }

    const visualViewport = window.visualViewport

    document.addEventListener('focusin', updateKeyboardState)
    document.addEventListener('focusout', updateKeyboardState)
    window.addEventListener('resize', updateKeyboardState)
    visualViewport?.addEventListener('resize', updateKeyboardState)
    visualViewport?.addEventListener('scroll', updateKeyboardState)
    updateKeyboardState()

    return () => {
      window.cancelAnimationFrame(frame)
      document.removeEventListener('focusin', updateKeyboardState)
      document.removeEventListener('focusout', updateKeyboardState)
      window.removeEventListener('resize', updateKeyboardState)
      visualViewport?.removeEventListener('resize', updateKeyboardState)
      visualViewport?.removeEventListener('scroll', updateKeyboardState)
    }
  }, [])

  return (
    <main className={shellClassName}>
      <Outlet />
      <BottomNav />
    </main>
  )
}

function getVisualViewportHeight() {
  return window.visualViewport?.height ?? window.innerHeight
}

function getVisualViewportOffsetTop() {
  return window.visualViewport?.offsetTop ?? 0
}

function isTextInputElement(element: Element | null) {
  if (!element) {
    return false
  }

  if (element instanceof HTMLTextAreaElement) {
    return true
  }
  if (element instanceof HTMLInputElement) {
    const nonTextTypes = new Set(['button', 'checkbox', 'color', 'file', 'hidden', 'image', 'radio', 'range', 'reset', 'submit'])
    return !nonTextTypes.has(element.type)
  }

  return element instanceof HTMLElement && element.isContentEditable
}
