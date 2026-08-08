import {
  Home,
  CircleDot,
  Users,
  MessageSquare,
  Settings,
  Flame,
  ScrollText,
  UserPlus,
  History,
  Sparkles,
  type LucideIcon,
} from 'lucide-react'

export interface NavChild {
  title: string
  description: string
  path: string
  icon: LucideIcon
}

export interface NavItem {
  label: string
  path: string
  icon: LucideIcon
  children?: NavChild[]
}

/** 顶部导航配置（Dify 式 mega-dropdown，参考 reference/导航栏.png） */
export const navItems: NavItem[] = [
  { label: '主殿', path: '/', icon: Home },
  {
    label: '金丹阁',
    path: '/pills',
    icon: CircleDot,
    children: [
      { title: '全部金丹', description: '浏览阁中已成之丹，观语言模式万千', path: '/pills', icon: CircleDot },
      { title: '开炉炼制', description: '以火为引，炼一枚新的语言模式金丹', path: '/pills?action=new', icon: Flame },
      { title: '丹方典籍', description: '查阅历代丹方，悟炼制之理', path: '/pills?view=recipes', icon: ScrollText },
    ],
  },
  {
    label: '道人府',
    path: '/agents',
    icon: Users,
    children: [
      { title: '全部道人', description: '府中道人名录，各司其职', path: '/agents', icon: Users },
      { title: '延请道人', description: '新请一位道人入府，授其金丹', path: '/agents?action=new', icon: UserPlus },
      { title: '服用金丹', description: '为道人绑定金丹，化性转型', path: '/agents?view=bind', icon: Sparkles },
    ],
  },
  {
    label: '论道',
    path: '/chat',
    icon: MessageSquare,
    children: [
      { title: '新的论道', description: '开一场新对话，与道人参详玄理', path: '/chat', icon: MessageSquare },
      { title: '论道旧录', description: '翻检历史会话，续当日未完之缘', path: '/chat?view=history', icon: History },
    ],
  },
  { label: '设置', path: '/settings', icon: Settings },
]
