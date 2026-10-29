export function cardStyle(active: boolean) {
  return {
    background: active ? 'rgba(59,130,246,0.12)' : 'transparent',
    borderColor: active ? 'rgba(96,165,250,0.56)' : 'transparent',
    color: 'var(--bm-text)',
  };
}
