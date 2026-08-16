export function notify(type, content) {
  window.dispatchEvent(new CustomEvent('app:toast', { detail: { type, content } }))
}
