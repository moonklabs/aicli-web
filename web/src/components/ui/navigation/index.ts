export { default as MobileNav } from './MobileNav.vue'
export { default as MobileTabBar } from './MobileTabBar.vue'
export { default as SwipeNavigation } from './SwipeNavigation.vue'

// 네비게이션 관련 타입 정의
export interface MenuItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  children?: MenuItem[]
}

export interface TabItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  badgeType?: 'primary' | 'success' | 'warning' | 'error' | 'info'
}

export interface SwipePage {
  id: string
  title?: string
  component?: any
  props?: Record<string, any>
  content?: string
  slot?: string
}