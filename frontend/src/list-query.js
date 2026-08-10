export function buildListPath(resource, pager, filters = {}) {
  const params = new URLSearchParams({
    page: String(pager.page),
    pageSize: String(pager.pageSize),
    sort: 'updatedAt',
    order: 'desc'
  })
  for (const [key, value] of Object.entries(filters)) {
    if (key !== 'advanced' && String(value || '').trim()) {
      params.set(key, String(value).trim())
    }
  }
  return `/${resource}?${params.toString()}`
}
