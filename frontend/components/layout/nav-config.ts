import {
  Home,
  CircleDot,
  Users,
  MessageSquare,
  Settings,
  History,
  FlaskConical,
  type LucideIcon,
} from 'lucide-react'

/**
 * Each nav entry is keyed by a dotted path into the `nav` namespace of
 * the i18n message catalog (see `frontend/messages/{locale}.json`).
 *
 *   labelKey   — top-level menu label
 *   titleKey   — child entry title (full text)
 *   descKey    — child entry one-line description
 *
 * This file owns structure (paths, icons, ordering) only — copy lives
 * entirely in the locale dictionaries.
 */
export interface NavChild {
  titleKey: string
  descKey: string
  path: string
  icon: LucideIcon
}

export interface NavItem {
  labelKey: string
  path: string
  icon: LucideIcon
  /** 额外视作本项激活的路径(如 /pills 下的子页面 /fusion) */
  activePaths?: string[]
  children?: NavChild[]
}

/**
 * 判断导航项是否激活：主路径或 activePaths 任一命中即激活。
 * `/` 精确匹配，其余按前缀匹配（pathname 须为去 locale 后的本地路径）。
 */
export function isNavItemActive(item: NavItem, pathname: string): boolean {
  const paths = [item.path, ...(item.activePaths ?? [])]
  return paths.some(path => path === '/' ? pathname === '/' : pathname.startsWith(path))
}

/** 顶部导航配置（Dify 式 mega-dropdown，参考 reference/导航栏.png） */
export const navItems: NavItem[] = [
  { labelKey: 'items.home.label', path: '/', icon: Home },
  {
    labelKey: 'items.pills.label',
    path: '/pills',
    icon: CircleDot,
    activePaths: ['/fusion'],
    children: [
      {
        titleKey: 'items.pills.children.all.title',
        descKey: 'items.pills.children.all.description',
        path: '/pills',
        icon: CircleDot,
      },
      {
        titleKey: 'items.pills.children.fusion.title',
        descKey: 'items.pills.children.fusion.description',
        path: '/fusion',
        icon: FlaskConical,
      },
    ],
  },
  {
    labelKey: 'items.agents.label',
    path: '/agents',
    icon: Users,
    children: [
      {
        titleKey: 'items.agents.children.all.title',
        descKey: 'items.agents.children.all.description',
        path: '/agents',
        icon: Users,
      },
    ],
  },
  {
    labelKey: 'items.chat.label',
    path: '/chat',
    icon: MessageSquare,
    children: [
      {
        titleKey: 'items.chat.children.new.title',
        descKey: 'items.chat.children.new.description',
        path: '/chat',
        icon: MessageSquare,
      },
      {
        titleKey: 'items.chat.children.history.title',
        descKey: 'items.chat.children.history.description',
        path: '/chat?view=history',
        icon: History,
      },
    ],
  },
  { labelKey: 'items.settings.label', path: '/settings', icon: Settings },
]
