export function phoneDialValue(phone) {
  const value = String(phone || '').trim()
  if (!/^\+?[0-9][0-9 -]*$/.test(value)) return ''
  const normalized = value.replace(/[ -]/g, '')
  return /^\+?[0-9]{3,20}$/.test(normalized) ? normalized : ''
}
