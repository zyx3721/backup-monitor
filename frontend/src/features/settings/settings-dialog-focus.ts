export function focusTextAreaAtEnd(
  textarea: HTMLTextAreaElement | null,
  fallbackInput: HTMLInputElement | null
) {
  if (textarea && !textarea.disabled) {
    textarea.focus();
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
    return;
  }
  if (!fallbackInput || fallbackInput.disabled) return;
  fallbackInput.focus();
  fallbackInput.setSelectionRange(fallbackInput.value.length, fallbackInput.value.length);
}
