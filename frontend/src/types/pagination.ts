export interface PageInfo {
  next_cursor: string | null
  has_more: boolean
}

export interface Page<T> {
  items: T[]
  page: PageInfo
}
