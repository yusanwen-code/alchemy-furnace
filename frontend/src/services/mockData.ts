/**
 * Mock 数据 - 前端独立演示使用
 * 包含金丹、丹方、道人、会话等示例数据
 */

import type { Pill, Recipe, Agent, ChatSession, ChatMessage } from './types'

// ========== 金丹 Mock 数据 ==========
export const mockPills: Pill[] = [
  {
    id: 1,
    name: '九转还魂丹',
    description: '汇聚修行界顶级功法与心得，可助道人突破境界瓶颈。包含各大门派的核心修炼法门、心法口诀及实战技巧。',
    status: 'refined',
    vector_count: 1280,
    created_at: '2024-12-01T10:30:00Z',
    updated_at: '2024-12-05T14:20:00Z',
  },
  {
    id: 2,
    name: '太乙紫金丹',
    description: '汇集道家典籍精华，涵盖道德经、庄子、列子等经典著作的深度解读与注释。',
    status: 'refined',
    vector_count: 856,
    created_at: '2024-12-03T08:15:00Z',
    updated_at: '2024-12-06T16:45:00Z',
  },
  {
    id: 3,
    name: '七魄归元丹',
    description: '炼制中... 收录现代项目管理与团队协作方法论，结合敏捷开发与精益思想。',
    status: 'refining',
    vector_count: 342,
    created_at: '2024-12-10T09:00:00Z',
    updated_at: '2024-12-10T11:30:00Z',
  },
  {
    id: 4,
    name: '混元无极丹',
    description: '融合多种编程语言的设计模式与最佳实践，从筑基到大乘的修行指南。',
    status: 'refined',
    vector_count: 2341,
    created_at: '2024-11-20T14:00:00Z',
    updated_at: '2024-12-01T09:15:00Z',
  },
  {
    id: 5,
    name: '玄黄造化丹',
    description: '炼丹失败... 尝试融合跨领域知识的实验性金丹。',
    status: 'failed',
    vector_count: 0,
    created_at: '2024-12-08T16:20:00Z',
    updated_at: '2024-12-09T10:00:00Z',
  },
]

// ========== 丹方 Mock 数据 ==========
export const mockRecipes: Record<number, Recipe[]> = {
  1: [
    { id: 1, pill_id: 1, filename: '修仙功法总纲.docx', file_type: 'docx', file_size: 2560000, file_path: '/uploads/1/修仙功法总纲.docx', extract_status: 'completed', chunk_count: 128, created_at: '2024-12-01T11:00:00Z' },
    { id: 2, pill_id: 1, filename: '心法口诀集.md', file_type: 'md', file_size: 512000, file_path: '/uploads/1/心法口诀集.md', extract_status: 'completed', chunk_count: 64, created_at: '2024-12-01T11:30:00Z' },
    { id: 3, pill_id: 1, filename: '门派比武记录.xlsx', file_type: 'xlsx', file_size: 1280000, file_path: '/uploads/1/门派比武记录.xlsx', extract_status: 'completed', chunk_count: 45, created_at: '2024-12-02T09:00:00Z' },
    { id: 4, pill_id: 1, filename: '丹药炼制心得.pdf', file_type: 'pdf', file_size: 4800000, file_path: '/uploads/1/丹药炼制心得.pdf', extract_status: 'completed', chunk_count: 200, created_at: '2024-12-03T14:00:00Z' },
    { id: 5, pill_id: 1, filename: '修炼日志.txt', file_type: 'txt', file_size: 25600, file_path: '/uploads/1/修炼日志.txt', extract_status: 'completed', chunk_count: 12, created_at: '2024-12-04T10:00:00Z' },
  ],
  2: [
    { id: 6, pill_id: 2, filename: '道德经注释.pdf', file_type: 'pdf', file_size: 8900000, file_path: '/uploads/2/道德经注释.pdf', extract_status: 'completed', chunk_count: 350, created_at: '2024-12-03T08:30:00Z' },
    { id: 7, pill_id: 2, filename: '庄子今译.docx', file_type: 'docx', file_size: 3200000, file_path: '/uploads/2/庄子今译.docx', extract_status: 'completed', chunk_count: 180, created_at: '2024-12-03T10:00:00Z' },
    { id: 8, pill_id: 2, filename: '列子集释.txt', file_type: 'txt', file_size: 1200000, file_path: '/uploads/2/列子集释.txt', extract_status: 'completed', chunk_count: 156, created_at: '2024-12-04T09:00:00Z' },
    { id: 9, pill_id: 2, filename: '道家思想史.md', file_type: 'md', file_size: 890000, file_path: '/uploads/2/道家思想史.md', extract_status: 'completed', chunk_count: 98, created_at: '2024-12-05T14:00:00Z' },
  ],
  3: [
    { id: 10, pill_id: 3, filename: '敏捷开发指南.docx', file_type: 'docx', file_size: 2100000, file_path: '/uploads/3/敏捷开发指南.docx', extract_status: 'completed', chunk_count: 95, created_at: '2024-12-10T09:30:00Z' },
    { id: 11, pill_id: 3, filename: '团队协作手册.pdf', file_type: 'pdf', file_size: 4500000, file_path: '/uploads/3/团队协作手册.pdf', extract_status: 'extracting', chunk_count: 0, created_at: '2024-12-10T10:00:00Z' },
    { id: 12, pill_id: 3, filename: '精益思想.md', file_type: 'md', file_size: 780000, file_path: '/uploads/3/精益思想.md', extract_status: 'pending', chunk_count: 0, created_at: '2024-12-10T11:00:00Z' },
  ],
  4: [
    { id: 13, pill_id: 4, filename: '设计模式之道.pdf', file_type: 'pdf', file_size: 12000000, file_path: '/uploads/4/设计模式之道.pdf', extract_status: 'completed', chunk_count: 520, created_at: '2024-11-21T10:00:00Z' },
    { id: 14, pill_id: 4, filename: 'Go语言修仙录.docx', file_type: 'docx', file_size: 5600000, file_path: '/uploads/4/Go语言修仙录.docx', extract_status: 'completed', chunk_count: 380, created_at: '2024-11-25T14:00:00Z' },
    { id: 15, pill_id: 4, filename: 'Python炼丹术.md', file_type: 'md', file_size: 3200000, file_path: '/uploads/4/Python炼丹术.md', extract_status: 'completed', chunk_count: 245, created_at: '2024-11-28T09:00:00Z' },
    { id: 16, pill_id: 4, filename: 'Rust铸器指南.pdf', file_type: 'pdf', file_size: 8900000, file_path: '/uploads/4/Rust铸器指南.pdf', extract_status: 'completed', chunk_count: 456, created_at: '2024-12-01T08:30:00Z' },
    { id: 17, pill_id: 4, filename: '代码重构心经.txt', file_type: 'txt', file_size: 450000, file_path: '/uploads/4/代码重构心经.txt', extract_status: 'completed', chunk_count: 89, created_at: '2024-12-01T09:00:00Z' },
  ],
}

// ========== 道人 Mock 数据 ==========
export const mockAgents: Agent[] = [
  {
    id: 1,
    name: '太虚真人',
    avatar: '',
    personality: '你是一位博学多才的道家高人，精通各类修行功法与心法。回答问题时喜欢引用道家经典，语气温和而充满智慧。你会用通俗易懂的方式解释复杂的修炼概念，偶尔穿插一些道家典故。',
    model_name: 'gpt-4o',
    status: 'active',
    created_at: '2024-11-15T08:00:00Z',
  },
  {
    id: 2,
    name: '玄天机子',
    avatar: '',
    personality: '你是一位擅长推演天机的道人，思维缜密，逻辑清晰。回答问题时条理分明，善于从多个角度分析问题。你会在回答末尾给出简短的总结，如同天机推演的结果。',
    model_name: 'deepseek-chat',
    status: 'active',
    created_at: '2024-11-18T10:30:00Z',
  },
  {
    id: 3,
    name: '紫霄仙子',
    avatar: '',
    personality: '你是一位温婉如玉的女修，精通丹药炼制与草药知识。回答问题时语言优美，如同诗词。你对丹药炼制有独到见解，能够详细解答各类丹方相关的问题。',
    model_name: 'gpt-4o',
    status: 'active',
    created_at: '2024-11-22T14:00:00Z',
  },
  {
    id: 4,
    name: '无极散人',
    avatar: '',
    personality: '你是一位不拘小节的散修，性格豪爽直率。回答问题开门见山，不拖泥带水。你喜欢用生活中的比喻来解释修行道理，让听者恍然大悟。',
    model_name: 'qwen-turbo',
    status: 'inactive',
    created_at: '2024-12-01T09:00:00Z',
  },
]

// ========== Agent-Pill 绑定关系 ==========
export const mockAgentPills: Record<number, number[]> = {
  1: [1, 2, 4],
  2: [2, 4],
  3: [1, 2],
  4: [],
}

// ========== 会话 Mock 数据 ==========
export const mockSessions: ChatSession[] = [
  { id: 1, agent_id: 1, title: '修仙功法探讨', created_at: '2024-12-08T10:00:00Z', updated_at: '2024-12-08T11:30:00Z' },
  { id: 2, agent_id: 1, title: '心法修炼疑问', created_at: '2024-12-09T14:00:00Z', updated_at: '2024-12-09T15:20:00Z' },
  { id: 3, agent_id: 2, title: '设计模式推演', created_at: '2024-12-10T09:00:00Z', updated_at: '2024-12-10T10:45:00Z' },
  { id: 4, agent_id: 3, title: '丹药炼制请教', created_at: '2024-12-10T11:00:00Z', updated_at: '2024-12-10T12:00:00Z' },
]

// ========== 消息 Mock 数据 ==========
export const mockMessages: Record<number, ChatMessage[]> = {
  1: [
    { id: 1, session_id: 1, role: 'user', content: '太虚真人，请问九转还魂丹中收录的《修仙功法总纲》主要讲了哪些内容？', created_at: '2024-12-08T10:00:00Z' },
    {
      id: 2, session_id: 1, role: 'assistant',
      content: '道友客气了。这《修仙功法总纲》乃是我金丹阁中的上乘之作，主要涵盖了以下几个方面的修炼精髓：\n\n## 一、筑基篇\n\n筑基乃是修仙之本。功法总纲中详细阐述了如何打通奇经八脉，凝聚丹田之气。其核心要义在于**静心凝神，气沉丹田**，每日卯时面对东方吐纳，吸收天地灵气。\n\n## 二、心法篇\n\n心法的修炼重在悟性。总纲中收录了"清心诀"、"凝神咒"等七部心法，每一部都配有详细的修炼步骤和注意事项。\n\n## 三、实战篇\n\n包括身法、剑法、符箓之术等实战技巧。特别强调**以意御气，以气御剑**的境界。\n\n道友若是对其中某一部分感兴趣，老夫可以详细为你解读。',
      sources: [
        { content: '筑基乃是修仙之本，需打通奇经八脉...', score: 0.95, metadata: { filename: '修仙功法总纲.docx', page: 12 } },
        { content: '心法修炼重在悟性，收录清心诀七部...', score: 0.88, metadata: { filename: '心法口诀集.md', page: 3 } },
      ],
      created_at: '2024-12-08T10:01:00Z',
    },
    { id: 3, session_id: 1, role: 'user', content: '那筑基篇中提到的卯时吐纳，具体应该怎么操作呢？', created_at: '2024-12-08T10:05:00Z' },
    {
      id: 4, session_id: 1, role: 'assistant',
      content: '好问道友！这卯时吐纳法乃是筑基的根本法门，操作得当可事半功倍。具体步骤如下：\n\n**时辰选择**：每日卯时（早上5-7点），此时阳气初生，最为适宜。\n\n**姿势要领**：\n1. 面东而立或端坐\n2. 双脚与肩同宽，膝盖微屈\n3. 双手结印，置于丹田\n4. 舌抵上颚，闭目凝神\n\n**吐纳流程**：\n- 先以鼻深吸一口气，想象天地灵气从百会穴灌入\n- 屏息三息，让灵气在体内流转\n- 再以口缓缓呼出浊气\n- 如此反复九九八十一周天\n\n> **注意**：初学者可从九周天开始，循序渐进，切勿贪功冒进。',
      sources: [
        { content: '卯时吐纳法：面东而立，双脚与肩同宽...', score: 0.97, metadata: { filename: '修仙功法总纲.docx', page: 15 } },
      ],
      created_at: '2024-12-08T10:06:00Z',
    },
  ],
  2: [
    { id: 5, session_id: 2, role: 'user', content: '真人，我最近修炼心法时总感觉心神不宁，有什么办法吗？', created_at: '2024-12-09T14:00:00Z' },
    {
      id: 6, session_id: 2, role: 'assistant',
      content: '道友这是遇到了修炼中的**心魔劫**，不必惊慌，这是每个修行者必经之路。\n\n心神不宁的原因通常有三：\n\n1. **杂念未除** - 修炼前心事未了，导致灵台不净\n2. **急于求成** - 功法进阶太快，心境跟不上修为\n3. **外魔入侵** - 修炼环境不佳，或受到外界干扰\n\n建议道友：\n\n- 先暂停功法修炼，转修"清心诀"七日\n- 每日以温水沐浴，净化身心\n- 可于子时（23:00-1:00）静坐冥想，此时阴气最盛，最易静心\n\n若七日后仍未好转，建议来金丹阁寻我，我可为你把脉诊断。',
      sources: [
        { content: '心魔劫的应对方法：暂停功法，转修清心诀...', score: 0.92, metadata: { filename: '心法口诀集.md', page: 28 } },
      ],
      created_at: '2024-12-09T14:02:00Z',
    },
  ],
  3: [
    { id: 7, session_id: 3, role: 'user', content: '机子道友，请问观察者模式在修仙系统中有什么应用场景？', created_at: '2024-12-10T09:00:00Z' },
    {
      id: 8, session_id: 3, role: 'assistant',
      content: '善哉此问！观察者模式在修仙界可谓无处不在，老夫为你推演几个典型的应用场景：\n\n## 1. 灵兽监控系统\n\n当灵兽的状态发生变化（如境界突破、受伤、饥饿）时，自动通知主人：\n\n```python\nclass SpiritualBeast(Subject):\n    def level_up(self):\n        self.state = "突破中"\n        self.notify_observers()\n\nclass Master(Observer):\n    def update(self, beast):\n        print(f"灵兽 {beast.name} 状态变更: {beast.state}")\n```\n\n## 2. 天劫预警系统\n\n修士渡劫时，多个防御阵法作为观察者，一旦劫云聚集即刻启动：\n\n## 3. 丹药炼制监控\n\n炼丹炉的温度、火候变化实时通知丹师，以便及时调整：\n\n此模式的精髓在于**松耦合**——主体无需关心观察者的具体实现，只需在状态变更时发出通知即可。',
      sources: [
        { content: '观察者模式：定义对象间的一对多依赖关系...', score: 0.94, metadata: { filename: '设计模式之道.pdf', page: 45 } },
      ],
      created_at: '2024-12-10T09:01:00Z',
    },
  ],
  4: [
    { id: 9, session_id: 4, role: 'user', content: '仙子，我想学习炼制筑基丹，需要什么材料？', created_at: '2024-12-10T11:00:00Z' },
    {
      id: 10, session_id: 4, role: 'assistant',
      content: '道友有此向道之心，甚好！筑基丹虽是入门丹药，却是修仙之路的基石，万万马虎不得。\n\n## 筑基丹方\n\n**主料**：\n- 百年灵芝 × 3（需采自灵气充沛的深山）\n- 天山雪莲 × 1（花瓣完整，色泽洁白）\n- 朱砂 × 5钱（以辰州朱砂为佳）\n\n**辅料**：\n- 清晨露水 × 1盏（需在日出前采集）\n- 无根水 × 半盏（雨水或雪水，未沾地气者）\n\n## 炼制步骤\n\n1. **净炉** - 以文火温炉三刻，驱除炉中杂气\n2. **入料** - 先下灵芝，待其化液后再入雪莲\n3. **调火** - 武火煮沸后转文火慢炖七七四十九息\n4. **凝丹** - 最后加入朱砂，以真火催动凝丹\n\n> ⚠️ **注意**：筑基丹一人一生只可服用一颗，多食无益。炼制时切忌心浮气躁，否则丹药品质大减。',
      sources: [
        { content: '筑基丹方：百年灵芝三株，天山雪莲一朵...', score: 0.96, metadata: { filename: '丹药炼制心得.pdf', page: 8 } },
      ],
      created_at: '2024-12-10T11:02:00Z',
    },
  ],
}

// ========== 可用模型列表 ==========
export const mockModels = [
  { id: 'gpt-4o', name: 'GPT-4o', description: 'OpenAI 旗舰模型', provider: 'openai' },
  { id: 'gpt-4o-mini', name: 'GPT-4o Mini', description: '轻量快速', provider: 'openai' },
  { id: 'deepseek-chat', name: 'DeepSeek-V3', description: '深度求索', provider: 'deepseek' },
  { id: 'deepseek-reasoner', name: 'DeepSeek-R1', description: '推理增强', provider: 'deepseek' },
  { id: 'qwen-turbo', name: '通义千问 Turbo', description: '阿里通义', provider: 'aliyun' },
  { id: 'qwen-plus', name: '通义千问 Plus', description: '增强版', provider: 'aliyun' },
]
