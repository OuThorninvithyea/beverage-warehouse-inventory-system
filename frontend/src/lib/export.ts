export function exportToCSV<T extends Record<string, any>>(
  filename: string,
  rows: T[],
  headers?: { key: keyof T; label: string }[],
) {
  if (!rows || rows.length === 0) return

  let csvContent = ''

  if (headers && headers.length > 0) {
    const headerRow = headers.map((h) => `"${String(h.label).replace(/"/g, '""')}"`).join(',')
    csvContent += headerRow + '\r\n'

    for (const row of rows) {
      const line = headers
        .map((h) => {
          const val = row[h.key]
          if (val === null || val === undefined) return '""'
          return `"${String(val).replace(/"/g, '""')}"`
        })
        .join(',')
      csvContent += line + '\r\n'
    }
  } else {
    const keys = Object.keys(rows[0])
    const headerRow = keys.map((k) => `"${k.replace(/"/g, '""')}"`).join(',')
    csvContent += headerRow + '\r\n'

    for (const row of rows) {
      const line = keys
        .map((k) => {
          const val = row[k]
          if (val === null || val === undefined) return '""'
          return `"${String(val).replace(/"/g, '""')}"`
        })
        .join(',')
      csvContent += line + '\r\n'
    }
  }

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.setAttribute('href', url)
  link.setAttribute('download', `${filename}.csv`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
