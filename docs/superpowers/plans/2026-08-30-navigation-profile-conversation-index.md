# 导航、个人头像与会话目录重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 精简金丹阁与道人府导航，补齐用户头像设置，并用后端权威身份数据重构“对谈 / 围炉论道”会话目录和群聊主题管理。

**Architecture:** 继续使用现有 `/api/v1/chat/sessions` 扁平会话接口，但让单聊响应直接携带道人名称、头像和状态，并让群聊创建原子接收可选主题。前端以纯函数构造“道人 → 单聊会话”分组，桌面侧栏与移动端 Sheet 共用同一目录组件；丹房旧录、聊天页头和消息气泡统一消费会话身份字段，UUID 永不作为可见名称兜底。

**Tech Stack:** Go 1.24、Gin、GORM、SQLite/PostgreSQL/MySQL、Next.js 16、React 19、TypeScript 5.7、Tailwind CSS 4、next-intl、Vitest、Testing Library

**Spec:** `docs/superpowers/specs/2026-08-30-navigation-profile-conversation-index-design.md`

## Global Constraints

- 保持 `/pills`、`/fusion`、`/agents`、`/chat?session=<UUID>` 路由不变。
- 不删除道人列表页内部的创建、搜索、筛选、详情编辑和服丹能力。
- 不新增头像上传、裁剪或文件存储服务。
- 头像 URL 最长 2048 字符；data URI 最长 1,500,000 字符，只允许 PNG、JPEG、WebP、GIF Base64 数据。
- 群聊主题去除首尾空格后最多 200 个 Unicode 字符；空主题表示等待首次问答自动命名。
- UUID 只允许用于 key、路由和 API 请求，不得渲染到任何用户可见文案。
- 单聊不新增手动主题输入；现有 SSE、群成员编排和历史只读规则保持不变。
- `frontend/messages/zh-CN.json` 与 `frontend/messages/en.json` 的新增、删除键必须同步。
- 每个任务严格执行：失败测试 → 确认失败原因 → 最小实现 → 目标测试通过 → 提交。
- 工作区已有用户修改；每次只暂存本任务列出的文件，不得使用 `git add .`。

---

## 文件结构与职责

- 后端身份契约集中在 `handler/chat/impl.go`，群聊标题规则集中在 `chat_service.go`。
- 后端头像格式规则集中到 `internal/util/avatar`，用户和道人 handler 只负责映射各自错误码。
- 前端头像纯校验放在 `lib/avatar-validation.ts`，不依赖 React Context。
- 前端会话排序与分组放在 `lib/session-presentation.ts`，展示放在 `components/chat/conversation-directory.tsx`。
- 丹房旧录和群标题编辑各自拆成小组件，避免继续扩大 `chat-view.tsx` 和首页文件。
- 所有 API 字段名只定义一次：Go DTO 使用 `agent_name` / `agent_avatar`，TypeScript `ChatSession` 使用同名字段。

---

### Task 1: 让会话响应携带单聊道人真实身份

**Files:**
- Modify: `backend/go/server/http/gateway/web/handler/chat/impl.go:32-92`
- Modify: `backend/go/server/http/gateway/web/handler/chat/impl_sse_chat_test.go:420-451`
- Modify: `backend/go/server/http/gateway/web/handler/chat/impl_create_session_test.go`
- Modify: `docs/api.md`

**Interfaces:**
- Produces JSON: `agent_name: string`、`agent_avatar: string`、现有 `agent_status: string`。
- Applies to: 单聊列表、直接读取、创建响应，因为三条路径共用 `toSessionResponse`。
- Group rule: 群聊三个单聊身份字段均为空或省略，成员仍来自 `members`。

- [ ] **Step 1: 写 DTO 失败测试**

```go
func TestSessionResponseIncludesSingleAgentIdentity(t *testing.T) {
	agentID := uint(7)
	agentUID := uuid.New()
	session := &model.ChatSession{
		UUID: uuid.New(), Type: model.SessionTypeSingle, AgentID: &agentID,
		Agent: model.DaoAgent{UUID: agentUID, Name: "太上老君", Avatar: "https://example.com/laojun.png", Status: "inactive"},
	}
	response := toSessionResponse(session)
	if response.AgentID != agentUID.String() || response.AgentName != "太上老君" {
		t.Fatalf("identity = %+v", response)
	}
	if response.AgentAvatar != "https://example.com/laojun.png" || response.AgentStatus != "inactive" {
		t.Fatalf("avatar/status = %+v", response)
	}
}
```

另测 group 响应的 `AgentID`、`AgentName`、`AgentAvatar` 均为空。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd backend/go && go test ./server/http/gateway/web/handler/chat -run 'TestSessionResponse' -v`

Expected: FAIL，`SessionResponse` 缺少新字段。

- [ ] **Step 3: 实现单聊身份映射**

DTO 增加：

```go
AgentName   string `json:"agent_name,omitempty"`
AgentAvatar string `json:"agent_avatar,omitempty"`
```

只在 `s.AgentID != nil` 时写入四个身份字段：

```go
if s.AgentID != nil {
	response.AgentID = s.Agent.UUID.String()
	response.AgentName = s.Agent.Name
	response.AgentAvatar = s.Agent.Avatar
	response.AgentStatus = s.Agent.Status
}
```

不要额外查询数据库；DAO 的读取和列表已经 `Preload("Agent")`，创建单聊也已赋值 `session.Agent`。

- [ ] **Step 4: 强化创建响应回归**

在 handler 创建测试新增单聊 stub，断言 HTTP 201 的 `data.agent_name` 和 `data.agent_avatar`。群聊现有成员响应测试继续通过。同步更新 `docs/api.md` 的会话 JSON 示例，使用字符串 UUID，并列出 `type`、`agent_id`、`agent_name`、`agent_avatar`、`agent_status`、`title`、`members`。

- [ ] **Step 5: 运行测试并提交**

Run: `cd backend/go && go test ./server/http/gateway/web/handler/chat ./internal/service/chat_service ./internal/dao`

Expected: PASS。

Commit:

```bash
git add backend/go/server/http/gateway/web/handler/chat docs/api.md
git commit -m "feat(chat): return single-agent session identity"
```

---

### Task 2: 原子保存围炉论道主题并统一标题边界

**Files:**
- Modify: `backend/go/internal/interface/service/chat.go:70-74`
- Modify: `backend/go/internal/service/chat_service/chat_service.go:278-292,453-490`
- Modify: `backend/go/internal/service/chat_service/group_service_test.go`
- Modify: `backend/go/internal/service/chat_service/group_orchestrator_test.go`
- Modify: `backend/go/server/http/gateway/web/handler/chat/impl_create_session.go:15-51`
- Modify: `backend/go/server/http/gateway/web/handler/chat/impl_create_session_test.go`
- Modify: every Go test stub found by `rg 'CreateGroupSession\(' backend/go`
- Modify: `docs/api.md`

**Interfaces:**
- Replaces: `CreateGroupSession(ctx, agentUIDs)`。
- Produces: `CreateGroupSession(ctx context.Context, agentUIDs []uuid.UUID, title string)`。
- Preserves: empty title triggers existing automatic title generation after first exchange。

- [ ] **Step 1: 写服务层标题失败测试**

```go
func TestCreateGroupSessionPersistsOptionalTitleAtomically(t *testing.T) {
	svc, chats, u1, u2, _ := newGroupTestSvc()
	session, err := svc.CreateGroupSession(context.Background(), []uuid.UUID{u1, u2}, "  丹道夜话  ")
	if err != nil {
		t.Fatal(err)
	}
	if session.Title != "丹道夜话" || chats.sessions[session.UUID.String()].Title != "丹道夜话" {
		t.Fatalf("title not persisted atomically: %+v", session)
	}
}

func TestCreateGroupSessionRejectsOverlongTitleBeforePersistence(t *testing.T) {
	svc, chats, u1, u2, _ := newGroupTestSvc()
	_, err := svc.CreateGroupSession(context.Background(), []uuid.UUID{u1, u2}, strings.Repeat("丹", 201))
	if err == nil || err.GetCode() != "service.chat.title_invalid" {
		t.Fatalf("error = %v", err)
	}
	if chats.groupSaveCalls != 0 || len(chats.sessions) != 0 {
		t.Fatal("invalid title must not persist session or members")
	}
}
```

把现有重命名边界测试从 31 改为 201，并新增正好 200 个 Unicode 字符成功。

- [ ] **Step 2: 运行服务测试并确认失败**

Run: `cd backend/go && go test ./internal/service/chat_service -run 'TestCreateGroupSession|TestUpdateSessionTitleValidation' -v`

Expected: FAIL，签名和 200 字边界尚未实现。

- [ ] **Step 3: 实现统一标题归一化**

```go
const sessionTitleMaxRunes = 200

func normalizeSessionTitle(title string, allowEmpty bool) (string, ierr.Error) {
	title = strings.TrimSpace(title)
	if title == "" && allowEmpty {
		return "", nil
	}
	if title == "" || utf8.RuneCountInString(title) > sessionTitleMaxRunes {
		return "", ierr.New(ierr.ErrorTypeInvalidRequest, "service.chat.title_invalid", "标题需为 1-200 个字符")
	}
	return title, nil
}
```

`UpdateSessionTitle` 使用 `allowEmpty=false`；`CreateGroupSession` 使用 `allowEmpty=true`，构造 `ChatSession` 时直接设置标题，再调用现有 `SaveGroupSession` 事务。自动生成标题仍截断 30 字，不改成 200。

- [ ] **Step 4: 写并实现 handler 转发测试**

Stub 记录 `gotTitle`，接口签名改为三参。用带空格的 `" 丹道夜话 "` 请求，断言 handler 原样交给 service，由 service 负责 trim。单聊仍调用 `CreateSession`，不消费客户端 title。

- [ ] **Step 5: 更新全部接口实现和测试桩**

执行 `rg -n 'CreateGroupSession\(' backend/go`。每个旧调用点显式传第三参；无主题的测试传 `""`。不要使用变参兼容旧签名。`docs/api.md` 增加 single 与 group 两种创建请求，明确 group 的 `title` 可选、最大 200 字、空值自动命名。

- [ ] **Step 6: 运行测试并提交**

Run: `cd backend/go && go test ./...`

Expected: PASS。

Commit:

```bash
git add backend/go/internal/interface/service/chat.go backend/go/internal/service/chat_service backend/go/server/http/gateway/web/handler/chat docs/api.md
git commit -m "feat(chat): save group topics atomically"
```

---

### Task 3: 同步前端会话身份并修复所有单聊名称表面

**Files:**
- Create: `frontend/components/home/recent-session-list.tsx`
- Create: `frontend/components/home/recent-session-list.test.tsx`
- Modify: `frontend/services/types.ts:280-294,386-395`
- Modify: `frontend/components/chat-message.tsx:50-105,120-170`
- Modify: `frontend/components/chat-message.test.tsx`
- Modify: `frontend/app/[locale]/page.tsx:103,317-361`
- Modify: `frontend/messages/zh-CN.json:185-191`
- Modify: `frontend/messages/en.json:185-191`

**Interfaces:**
- `ChatSession` adds optional `agent_name?: string`、`agent_avatar?: string`；保留 `agent_status`。
- Removes: 前端未被服务端返回的 `agent?: Agent` 字段。
- Produces: `RecentSessionList({sessions})`，只渲染最近四条。

- [ ] **Step 1: 写丹房旧录失败测试**

```tsx
it('shows authoritative single-agent names and never renders UUIDs', () => {
  const agentId = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
  render(<RecentSessionList sessions={[{
    id: '11111111-1111-4111-8111-111111111111',
    type: 'single', agent_id: agentId, agent_name: '太上老君', title: '炼丹之问',
    created_at: '2026-08-20T00:00:00Z', updated_at: '2026-08-21T00:00:00Z',
  }]} />)
  expect(screen.getByText('太上老君')).toBeInTheDocument()
  expect(document.body.textContent).not.toContain(agentId)
})
```

再测：身份缺失显示 `unknownAgent`，group 显示 `groupIdentity` 和成员数，二者都不显示 UUID。

- [ ] **Step 2: 写单聊消息气泡身份失败测试**

当前会话携带 `agent_name: '真实道号'`、`agent_avatar`，全局道人列表设为空，渲染没有 `message.agent_name` 的历史 assistant 消息。断言角色标签和头像 alt 是“真实道号”，不是 `assistantLabel`。

- [ ] **Step 3: 运行目标测试并确认失败**

Run: `cd frontend && pnpm test -- components/home/recent-session-list.test.tsx components/chat-message.test.tsx`

Expected: FAIL，新组件和身份字段尚不存在。

- [ ] **Step 4: 更新类型并实现统一身份优先级**

```ts
export interface ChatSession {
  id: string
  type?: 'single' | 'group'
  agent_id: string
  agent_name?: string
  agent_avatar?: string
  agent_status?: AgentStatus
  title?: string
  created_at: string
  updated_at: string
  members?: GroupMember[]
}
```

ChatMessage 使用 `message.agent_name || currentSession.agent_name || assistantLabel`，头像使用 `message.agent_avatar || agentProfile.avatar || memberAvatar || currentSession.agent_avatar`。群聊不得使用单聊的 session 身份。若 popover 需要简略 `Agent`，由当前会话的新字段构造，不再读取 `currentSession.agent`。

- [ ] **Step 5: 抽取并接入 RecentSessionList**

组件规则：单聊 badge 为 `session.agent_name || t('unknownAgent')`；群聊 badge 为 `t('groupIdentity', {count: session.members?.length ?? 0})`。HomePage 用该组件替换旧 `<ol>`。删除会插入 ID 的 `fallbackAgent`，新增中英文 `unknownAgent` 与 `groupIdentity`。

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- components/home/recent-session-list.test.tsx components/chat-message.test.tsx && pnpm typecheck`

Expected: PASS，且 `rg -n 'fallbackAgent|session\.agent|s\.agent' frontend` 不再命中会话身份展示代码。

Commit:

```bash
git add frontend/services/types.ts frontend/components/chat-message.tsx frontend/components/chat-message.test.tsx frontend/components/home frontend/app/'[locale]'/page.tsx frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "fix(chat): display authoritative daoist identities"
```

---

### Task 4: 统一后端头像契约并开放用户头像持久化

**Files:**
- Create: `backend/go/internal/util/avatar/avatar.go`
- Create: `backend/go/internal/util/avatar/avatar_test.go`
- Create: `backend/go/server/http/gateway/web/handler/user/impl_avatar_test.go`
- Modify: `backend/go/server/http/gateway/web/handler/agent/avatar.go`
- Modify: `backend/go/server/http/gateway/web/handler/user/impl.go:47-90`
- Modify: `backend/go/model/models.go:564-571`
- Modify: `backend/go/internal/dao/migrate.go:51-57`
- Modify: `backend/go/internal/dao/migrate_smoke_test.go`
- Test: `backend/go/server/http/gateway/web/handler/agent/impl_avatar_test.go`
- Modify: `docs/api.md`

**Interfaces:**
- Produces: `avatar.Validate(string) error`、`avatar.MaxURLLen`、`avatar.MaxDataURILen`。
- Preserves: 道人接口错误码 `handler.agent.avatar_validate`。
- Produces: 用户头像非法时错误码 `handler.user.avatar_validate`。

- [ ] **Step 1: 写共享校验器的失败测试**

在 `avatar_test.go` 写表驱动测试：空值、完整 HTTP/HTTPS URL 和四种允许的 data URI 必须成功；相对路径、可执行协议、内嵌凭据、SVG、非 Base64 和非法字符必须失败。长度边界单独测试 `MaxURLLen` 与 `MaxDataURILen`。

```go
func TestValidate(t *testing.T) {
	valid := []string{"", "https://cdn.example.com/a.png", "data:image/png;base64,AAAA"}
	for _, value := range valid {
		if err := Validate(value); err != nil {
			t.Fatalf("Validate(%q) = %v, want nil", value, err)
		}
	}
	invalid := []string{"/a.png", "javascript:alert(1)", "data:image/svg+xml;base64,AAAA", "data:image/png;base64,@@@"}
	for _, value := range invalid {
		if err := Validate(value); err == nil {
			t.Fatalf("Validate(%q) = nil, want error", value)
		}
	}
}
```

- [ ] **Step 2: 运行共享测试并确认失败**

Run: `cd backend/go && go test ./internal/util/avatar -run TestValidate -v`

Expected: FAIL，因为 `internal/util/avatar` 包尚不存在。

- [ ] **Step 3: 实现共享校验器并让道人处理器调用它**

将现有 `handler/agent/avatar.go` 的 URL、MIME、Base64 和长度逻辑移动到新包。共享包返回不包含原始头像值的普通错误：

```go
package avatar

const (
	MaxURLLen     = 2048
	MaxDataURILen = 1_500_000
)

func Validate(value string) error {
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "data:image/") {
		return validateDataURI(value)
	}
	return validateURL(value)
}
```

`handler/agent/avatar.go` 只保留错误码包装：

```go
func validateAvatar(value string) errors.Error {
	if err := avatar.Validate(value); err != nil {
		return errors.New(errors.ErrorTypeInvalidRequest, "handler.agent.avatar_validate", err.Error())
	}
	return nil
}
```

- [ ] **Step 4: 写用户头像接口失败测试**

使用 SQLite 内存库迁移 `model.UserProfile`，挂载真实 `PUT /api/v1/user/profile`。断言合法 data URI 原样保存、超过 500 字符仍保存、空字符串清除；非法协议返回 HTTP 400 和 `handler.user.avatar_validate`，响应不得含 Base64 payload。同步扩展 `migrate_smoke_test.go`：`wantTables` 必须包含 `user_profile`，并用 `db.Migrator().ColumnTypes(&model.UserProfile{})` 断言 `avatar` 数据库类型为 `text`。

```go
status, body := putUserProfile(t, router, `{"display_name":"炉主","avatar":"data:image/png;base64,AAAA"}`)
if status != http.StatusOK {
	t.Fatalf("valid avatar status = %d, body = %v", status, body)
}
status, body = putUserProfile(t, router, `{"avatar":"javascript:alert(1)"}`)
if status != http.StatusBadRequest || body["error_code"] != "handler.user.avatar_validate" {
	t.Fatalf("invalid avatar response = %d %v", status, body)
}
```

- [ ] **Step 5: 实现用户校验与数据库列加宽**

删除 `UpdateRequest.Avatar` 上的 `binding:"omitempty,max=500"`；trim 后调用共享校验器。模型字段改为 `gorm:"type:text"`，并在 `columnTypeAlterations` 追加 `{"user_profile", "avatar", "text"}`。历史 SQL 目录已废弃，不新增 raw SQL 文件。`docs/api.md` 补充 `GET/PUT /api/v1/user/profile`，列出头像格式、长度和清空规则。

```go
if body.Avatar != nil {
	value := strings.TrimSpace(*body.Avatar)
	if err := avatar.Validate(value); err != nil {
		return response.InvalidParams, nil, ierr.New(ierr.ErrorTypeInvalidRequest, "handler.user.avatar_validate", err.Error())
	}
	updates["avatar"] = value
}
```

- [ ] **Step 6: 运行测试并提交**

Run: `cd backend/go && go test ./internal/util/avatar ./server/http/gateway/web/handler/user ./server/http/gateway/web/handler/agent ./internal/dao`

Expected: PASS。

Commit:

```bash
git add backend/go/internal/util/avatar backend/go/server/http/gateway/web/handler/agent/avatar.go backend/go/server/http/gateway/web/handler/user backend/go/model/models.go backend/go/internal/dao/migrate.go backend/go/internal/dao/migrate_smoke_test.go docs/api.md
git commit -m "feat(profile): validate and persist user avatars"
```

---

### Task 5: 在“我的简介”中编辑和预览头像

**Files:**
- Create: `frontend/lib/avatar-validation.ts`
- Create: `frontend/lib/avatar-validation.test.ts`
- Modify: `frontend/lib/avatar-url.ts`
- Modify: `frontend/lib/avatar-url.test.ts`
- Modify: `frontend/hooks/use-agent-editor-flow.ts:17-49`
- Modify: `frontend/app/(main)/agents/page.tsx`
- Modify: `frontend/app/(main)/agents/detail/agent-detail.tsx`
- Modify: `frontend/components/settings/profile-panel.tsx`
- Modify: `frontend/components/settings/profile-panel.test.tsx`
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**
- Produces: `validateAvatarField(value): 'invalid' | 'tooLong' | undefined`。
- Produces: `avatarInputMaxLength(value): 2048 | 1500000`。
- Consumes: `EntityAvatar`、`UserProfile.avatar`、`updateProfile({display_name,bio,avatar})`。

- [ ] **Step 1: 写前端头像校验失败测试**

```ts
it.each([
  ['', undefined],
  ['https://cdn.example.com/a.png', undefined],
  ['data:image/png;base64,AAAA', undefined],
  ['/a.png', 'invalid'],
  ['javascript:alert(1)', 'invalid'],
  ['https://user:pass@example.com/a.png', 'invalid'],
  ['data:image/svg+xml;base64,AAAA', 'invalid'],
  ['data:image/png;base64,@@@', 'invalid'],
])('validates %s', (value, expected) => {
  expect(validateAvatarField(value)).toBe(expected)
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd frontend && pnpm test -- lib/avatar-validation.test.ts`

Expected: FAIL，模块不存在。

- [ ] **Step 3: 抽取共用校验并保持道人流程不变**

把 `use-agent-editor-flow.ts` 中的头像常量和两个纯函数移动到 `lib/avatar-validation.ts`，并补齐与 Go 一致的“URL 不得含用户名/密码”和“data URI payload 只能含 Base64 字符”规则。该模块直接使用 `new URL` 与 data URI 解析，不反向导入 `avatar-url.ts`。

原 hook、道人创建页和道人详情页都改为从新模块导入。`normalizeAvatarUrl` 调用 `validateAvatarField`，校验失败返回 `undefined`，成功时再返回规范 URL 或原 data URI，保证预览层也不会加载后端拒绝的值。

- [ ] **Step 4: 扩展 ProfilePanel 失败测试**

```ts
it('previews and saves the user avatar', async () => {
  td.userState.profile = { ...profileA, avatar: 'https://example.com/old.png' }
  const user = userEvent.setup()
  render(<ProfilePanel />)
  expect(screen.getByRole('img', { name: 'avatarPreviewAlt' })).toHaveAttribute('src', 'https://example.com/old.png')
  const input = screen.getByLabelText('avatarLabel')
  await user.clear(input)
  await user.type(input, 'https://example.com/new.png')
  await user.click(screen.getByRole('button', { name: 'save' }))
  expect(td.updateProfile).toHaveBeenCalledWith({ display_name: 'Yao', bio: 'Alchemist', avatar: 'https://example.com/new.png' })
})
```

再断言非法值零 API、图片加载失败回退首字、清空后提交 `avatar: ''`。现有 “saves the trimmed display name and bio” 用例的期望载荷同步增加 `avatar: ''`，不要删除旧的 name/bio 断言。

- [ ] **Step 5: 实现头像编辑区**

新增 `avatar` state；profile 引用变化时回填。保存前用共享校验器，保存载荷包含头像。使用 `EntityAvatar` 实时预览，输入框 `maxLength={avatarInputMaxLength(avatar)}`。新增中英文键：`avatarLabel`、`avatarPlaceholder`、`avatarHint`、`avatarPreviewAlt`、`avatarInvalid`、`avatarTooLong`。

```tsx
<EntityAvatar name={displayName.trim() || t('defaultUser')} src={avatar} size="lg" shape="circle" alt={t('avatarPreviewAlt')} />
<input id="profile-avatar" value={avatar} onChange={event => setAvatar(event.target.value)} maxLength={avatarInputMaxLength(avatar)} />
```

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- lib/avatar-validation.test.ts lib/avatar-url.test.ts components/settings/profile-panel.test.tsx app/\(main\)/agents/page.test.tsx app/\(main\)/agents/detail/agent-detail.test.tsx`

Expected: PASS。

Commit:

```bash
git add frontend/lib/avatar-validation.ts frontend/lib/avatar-validation.test.ts frontend/lib/avatar-url.ts frontend/lib/avatar-url.test.ts frontend/hooks/use-agent-editor-flow.ts frontend/app/'(main)'/agents frontend/components/settings/profile-panel.tsx frontend/components/settings/profile-panel.test.tsx frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(profile): edit and preview user avatar"
```

---

### Task 6: 精简金丹阁与道人府导航

**Files:**
- Create: `frontend/components/layout/nav-config.test.ts`
- Modify: `frontend/components/layout/nav-config.ts`
- Modify: `frontend/components/layout/navbar.tsx:29-40,92-95,198-205`
- Modify: `frontend/messages/zh-CN.json:14-70`
- Modify: `frontend/messages/en.json:14-70`

**Interfaces:**
- Produces: `isNavItemActive(item: NavItem, pathname: string): boolean`。
- Produces: 金丹阁子项 `/pills`、`/fusion`；道人府唯一子项 `/agents`。

- [ ] **Step 1: 写导航结构与激活失败测试**

```ts
it('keeps only the approved children', () => {
  const pills = navItems.find(item => item.path === '/pills')!
  const agents = navItems.find(item => item.path === '/agents')!
  expect(pills.children?.map(child => child.path)).toEqual(['/pills', '/fusion'])
  expect(agents.children?.map(child => child.path)).toEqual(['/agents'])
  expect(navItems.some(item => item.path === '/fusion')).toBe(false)
  expect(isNavItemActive(pills, '/fusion')).toBe(true)
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd frontend && pnpm test -- components/layout/nav-config.test.ts`

Expected: FAIL，仍有独立 `/fusion` 和多余子项。

- [ ] **Step 3: 修改导航结构和激活函数**

给 `NavItem` 增加 `activePaths?: string[]`，金丹阁设置 `activePaths: ['/fusion']`。删除独立顶级 fusion；金丹阁 children 只留 all 和 fusion；道人府只留 all。

```ts
export function isNavItemActive(item: NavItem, pathname: string): boolean {
  const paths = [item.path, ...(item.activePaths ?? [])]
  return paths.some(path => path === '/' ? pathname === '/' : pathname.startsWith(path))
}
```

Navbar 桌面和移动端都调用该函数。

- [ ] **Step 4: 同步中英文导航文案**

新增 `items.pills.children.fusion`，中文标题“融合金丹”，英文标题“Fuse pills”。删除 pills 的 new/recipes、agents 的 invite/bind 和顶级 items.fusion 文案。

- [ ] **Step 5: 运行测试并提交**

Run: `cd frontend && pnpm test -- components/layout/nav-config.test.ts && pnpm typecheck`

Expected: PASS。

Commit:

```bash
git add frontend/components/layout/nav-config.ts frontend/components/layout/nav-config.test.ts frontend/components/layout/navbar.tsx frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(nav): merge fusion into pill vault"
```

---

### Task 7: 建立会话分组纯函数和双 Tab 目录组件

**Files:**
- Create: `frontend/lib/session-presentation.ts`
- Create: `frontend/lib/session-presentation.test.ts`
- Create: `frontend/components/chat/conversation-directory.tsx`
- Create: `frontend/components/chat/conversation-directory.test.tsx`
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**
- Produces: `sessionKind(session): 'single' | 'group'`。
- Produces: `groupSingleSessions(sessions): SingleSessionGroup[]`。
- Produces: `ConversationDirectory({sessions,currentSessionId,onSelect})`。

- [ ] **Step 1: 写纯函数失败测试**

```ts
it('groups legacy and single sessions by agent and sorts newest first', () => {
  const groups = groupSingleSessions([
    session('old-a', undefined, 'agent-a', 'Alpha', '2026-08-20T00:00:00Z'),
    session('new-b', 'single', 'agent-b', 'Beta', '2026-08-23T00:00:00Z'),
    session('new-a', 'single', 'agent-a', 'Alpha', '2026-08-22T00:00:00Z'),
    groupSession,
  ])
  expect(groups.map(group => group.agentId)).toEqual(['agent-b', 'agent-a'])
  expect(groups[1].sessions.map(item => item.id)).toEqual(['new-a', 'old-a'])
})

it('never turns agent ids into visible names', () => {
  const groups = groupSingleSessions([session('x', 'single', 'secret-uuid', undefined, now)])
  expect(groups[0].agentName).toBe('')
})
```

- [ ] **Step 2: 运行纯函数测试并确认失败**

Run: `cd frontend && pnpm test -- lib/session-presentation.test.ts`

Expected: FAIL，模块不存在。

- [ ] **Step 3: 实现纯函数**

```ts
export interface SingleSessionGroup {
  agentId: string
  agentName: string
  agentAvatar?: string
  latestUpdatedAt: string
  sessions: ChatSession[]
}

export function sessionKind(session: ChatSession): 'single' | 'group' {
  return session.type === 'group' ? 'group' : 'single'
}
```

`groupSingleSessions` 先过滤 single/历史缺省类型，再按 `agent_id` 分组；每组会话按 `updated_at` 倒序，父组按最新子会话时间倒序。`agentName` 只取 `agent_name || ''`，禁止复制 `agent_id`。

- [ ] **Step 4: 写目录组件失败测试**

```tsx
it('defaults to single, groups by daoist, and opens the active group', () => {
  render(<ConversationDirectory sessions={sessions} currentSessionId="single-a-2" onSelect={onSelect} />)
  expect(screen.getByRole('tab', { name: 'tabs.single' })).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByRole('button', { name: /Alpha/ })).toHaveAttribute('aria-expanded', 'true')
  expect(screen.getByText('Alpha second')).toBeInTheDocument()
})
```

再覆盖：切换群聊 Tab、group 深链默认选中群聊、两个独立空状态、点击会话调用 `onSelect(id)`、父级收起和页面文本不含 agent UUID。

- [ ] **Step 5: 实现 ConversationDirectory 和文案**

Props 固定为：

```ts
export interface ConversationDirectoryProps {
  sessions: ChatSession[]
  currentSessionId?: string
  onSelect: (sessionId: string) => void
}
```

组件只消费 props，不请求 API、不读取 `AgentContext`。使用 `TopTabs`；当 `currentSessionId` 改变时，把 active Tab 同步为该会话类型，并自动展开其道人父级。父级按钮显示 `EntityAvatar`、`agentName || unknownAgent`、会话数和 `aria-expanded`。群聊显示主题、最多三个成员头像、成员数和更新时间。

在 `chatView.directory` 增加中英文键：`tabs.single`、`tabs.group`、`unknownAgent`、`untitledSingle`、`untitledGroup`、`singleCount`、`groupMeta`、`emptySingle`、`emptyGroup`。

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- lib/session-presentation.test.ts components/chat/conversation-directory.test.tsx`

Expected: PASS。

Commit:

```bash
git add frontend/lib/session-presentation.ts frontend/lib/session-presentation.test.ts frontend/components/chat frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(chat): add grouped conversation directory"
```

---

### Task 8: 将统一目录接入桌面侧栏和移动端 Sheet

**Files:**
- Modify: `frontend/app/(main)/chat/chat-view.tsx:254-260,425-530,554-580`
- Modify: `frontend/app/(main)/chat/chat-view.test.tsx`
- Modify: `frontend/app/(main)/chat/chat-view-context.test.tsx`

**Interfaces:**
- Consumes: `ConversationDirectory`、`ChatSession.agent_name`、`agent_avatar`。
- Preserves: `handleSelectSession` 继续导航到 `chatSessionHref(sessionId)`。

- [ ] **Step 1: 写 ChatView 集成失败测试**

把单聊 fixture 加上 `agent_name: 'Agent One'` 和头像。新增断言：即使 `agentState.agents=[]`，桌面目录父级和单聊页头仍显示 Agent One；切换“围炉论道”只见群聊；group 深链默认选中群聊；移动端 Sheet 使用同样目录结构。

- [ ] **Step 2: 运行集成测试并确认失败**

Run: `cd frontend && pnpm test -- app/\(main\)/chat/chat-view.test.tsx app/\(main\)/chat/chat-view-context.test.tsx`

Expected: FAIL，ChatView 仍使用重复平铺列表和道人列表查名。

- [ ] **Step 3: 替换桌面和移动端重复列表**

删除所有侧栏 `sessions.map` 以及 `getAgentName` / `getAgentInitial`。桌面与 Sheet 都渲染：

```tsx
<ConversationDirectory
  sessions={sessions}
  currentSessionId={currentSession?.id}
  onSelect={handleSelectSession}
/>
```

移动端 `onSelect` 包装函数先关闭 Sheet 再导航。

- [ ] **Step 4: 修复单聊页头身份**

```tsx
<EntityAvatar
  name={currentSession.agent_name || t('directory.unknownAgent')}
  src={currentSession.agent_avatar}
  size="sm"
  shape="circle"
/>
<p>{currentSession.agent_name || t('directory.unknownAgent')}</p>
<p>{currentSession.title || t('directory.untitledSingle')}</p>
```

不要再把会话标题当成道人名称，也不要把 `agent_id` 交给翻译函数。

- [ ] **Step 5: 更新深链 fixture 和回归断言**

创建响应与 `getSession` 单聊响应都加入新身份字段。断言规范 URL 重新挂载后页头仍显示服务端名称；群聊深链行为保持不变。

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- components/chat/conversation-directory.test.tsx app/\(main\)/chat/chat-view.test.tsx app/\(main\)/chat/chat-view-context.test.tsx && pnpm typecheck`

Expected: PASS。

Commit:

```bash
git add frontend/app/'(main)'/chat frontend/components/chat/conversation-directory.tsx frontend/components/chat/conversation-directory.test.tsx
git commit -m "feat(chat): integrate tabbed conversation directory"
```

---

### Task 9: 将围炉主题贯穿创建、失败保留与重试链路

**Files:**
- Modify: `frontend/services/types.ts:386-395`
- Modify: `frontend/services/chatService.ts:55-60`
- Modify: `frontend/contexts/ChatContext.tsx:385-390,485-497`
- Modify: `frontend/hooks/use-chat-launch-flow.ts`
- Modify: `frontend/hooks/use-chat-launch-flow.test.tsx`
- Modify: `frontend/app/(main)/chat/chat-view.tsx:831-1035`
- Modify: `frontend/app/(main)/chat/chat-view.test.tsx`
- Modify: `frontend/app/(main)/chat/chat-view-context.test.tsx`
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**
- Produces: `chatService.createGroupSession(memberAgentIds: string[], title?: string)`。
- Produces: `ChatContext.createGroupSession(memberAgentIds: string[], title?: string)`。
- Produces: `ChatLaunchFlow.launchGroup(agentIds: string[], title?: string)`。

- [ ] **Step 1: 写 launch flow 主题保留失败测试**

```ts
await act(async () => {
  expect(await result.current.launchGroup(selectedAgentIds, ' 丹道夜话 ')).toBe(false)
})
await act(async () => {
  expect(await result.current.retry()).toBe(true)
})
expect(createGroupSession).toHaveBeenNthCalledWith(1, ['agent-1', 'agent-2'], ' 丹道夜话 ')
expect(createGroupSession).toHaveBeenNthCalledWith(2, ['agent-1', 'agent-2'], ' 丹道夜话 ')
```

- [ ] **Step 2: 写创建弹窗失败测试**

群模式输入 `topicLabel`，选择两人，确认后断言创建函数收到成员和主题。第一次请求失败后主题输入仍保留；点击 retry 后仍携带同一主题。另测 201 字显示 `topicTooLong` 且零 API，空输入允许创建并传 `undefined`。

- [ ] **Step 3: 运行测试并确认失败**

Run: `cd frontend && pnpm test -- hooks/use-chat-launch-flow.test.tsx app/\(main\)/chat/chat-view.test.tsx`

Expected: FAIL，当前链路只有成员数组。

- [ ] **Step 4: 修改 service、context 和 launch flow**

```ts
export function createGroupSession(memberAgentIds: string[], title?: string): Promise<ChatSession> {
  return post<ChatSession>('/chat/sessions', {
    type: 'group', member_agent_ids: memberAgentIds, title: title?.trim() || undefined,
  })
}
```

`CreateSessionRequest` 注释改为“single 忽略 title；group 接受可选 title”。`LaunchRequest` 的 group 分支增加 `title?: string`。`launchGroup` 复制成员数组并保存主题；重试复用失败请求快照，不读取弹窗当前 state。

- [ ] **Step 5: 在群模式增加主题输入和文案**

AgentSelectModal 增加 `topic` state，仅 group 模式显示输入。主题 `onChange` 必须先调用 `onSelectionChange()` 清除旧的失败请求和错误，再更新 state；因此用户改主题后不能误重试旧主题。提交前用 `Array.from(topic.trim()).length` 校验 200 个 Unicode 字符。模式切换到 single 后，单聊提交绝不发送 topic。

新增中英文 `mode.topicLabel`、`topicPlaceholder`、`topicHint`、`topicTooLong`。

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- hooks/use-chat-launch-flow.test.tsx app/\(main\)/chat/chat-view.test.tsx app/\(main\)/chat/chat-view-context.test.tsx`

Expected: PASS。

Commit:

```bash
git add frontend/services/types.ts frontend/services/chatService.ts frontend/contexts/ChatContext.tsx frontend/hooks/use-chat-launch-flow.ts frontend/hooks/use-chat-launch-flow.test.tsx frontend/app/'(main)'/chat frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(chat): set topics when creating group sessions"
```

---

### Task 10: 支持围炉论道创建后重命名

**Files:**
- Create: `frontend/components/chat/group-topic-editor.tsx`
- Create: `frontend/components/chat/group-topic-editor.test.tsx`
- Modify: `frontend/app/(main)/chat/chat-view.tsx:554-570`
- Modify: `frontend/app/(main)/chat/chat-view.test.tsx`
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**
- Produces: `GroupTopicEditor({sessionId,title,onRename})`。
- Consumes: `onRename(sessionId, title): Promise<ChatSession | null>`。
- Success rule: 返回非 null 才关闭编辑态；失败保留草稿。

- [ ] **Step 1: 写重命名组件失败测试**

```tsx
it('trims and saves a valid topic', async () => {
  const onRename = vi.fn().mockResolvedValue({ ...groupSession, title: '新主题' })
  const user = userEvent.setup()
  render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
  await user.click(screen.getByRole('button', { name: 'rename' }))
  await user.clear(screen.getByLabelText('renameLabel'))
  await user.type(screen.getByLabelText('renameLabel'), '  新主题  ')
  await user.click(screen.getByRole('button', { name: 'saveRename' }))
  expect(onRename).toHaveBeenCalledWith('group-1', '新主题')
  expect(screen.queryByLabelText('renameLabel')).not.toBeInTheDocument()
})

it('keeps the draft open after a failed rename', async () => {
  const onRename = vi.fn().mockResolvedValue(null)
  const user = userEvent.setup()
  render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
  await user.click(screen.getByRole('button', { name: 'rename' }))
  await user.clear(screen.getByLabelText('renameLabel'))
  await user.type(screen.getByLabelText('renameLabel'), '仍要保存的主题')
  await user.click(screen.getByRole('button', { name: 'saveRename' }))
  expect(screen.getByLabelText('renameLabel')).toHaveValue('仍要保存的主题')
  expect(screen.getByRole('alert')).toHaveTextContent('renameError')
})
```

再覆盖空白、201 字零 API、Escape 取消、提交中禁止重复保存。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd frontend && pnpm test -- components/chat/group-topic-editor.test.tsx`

Expected: FAIL，组件不存在。

- [ ] **Step 3: 实现 GroupTopicEditor**

```ts
export interface GroupTopicEditorProps {
  sessionId: string
  title: string
  onRename: (sessionId: string, title: string) => Promise<ChatSession | null>
}
```

默认展示标题和编辑按钮；进入编辑态复制 `title` 到 draft。保存时 trim 并按 Unicode 字符数校验。`onRename` 返回非 null 才退出；null 时保留 draft 并显示错误。Enter 保存、Escape 取消，提交中禁用输入和按钮。

- [ ] **Step 4: 接入群聊页头**

```tsx
<GroupTopicEditor
  sessionId={currentSession.id}
  title={currentSession.title || t('directory.untitledGroup')}
  onRename={renameSession}
/>
```

群成员按钮保留；单聊页头不渲染重命名控件。

- [ ] **Step 5: 增加文案和 ChatView 回归**

在 `chatView.topic` 同步增加 `rename`、`renameLabel`、`saveRename`、`cancelRename`、`renameError`、`emptyError`、`tooLongError`。ChatView 测试断言：成功后页头和目录同时更新；失败后两处仍是旧标题，编辑框保留新草稿。

- [ ] **Step 6: 运行测试并提交**

Run: `cd frontend && pnpm test -- components/chat/group-topic-editor.test.tsx app/\(main\)/chat/chat-view.test.tsx hooks/use-chat-launch-flow.test.tsx && pnpm typecheck`

Expected: PASS。

Commit:

```bash
git add frontend/components/chat/group-topic-editor.tsx frontend/components/chat/group-topic-editor.test.tsx frontend/app/'(main)'/chat/chat-view.tsx frontend/app/'(main)'/chat/chat-view.test.tsx frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(chat): rename group discussion topics"
```

---

### Task 11: 全量验证与双端人工验收

**Files:**
- No planned source changes; this task only runs verification and records results.

**Interfaces:**
- Verifies all interfaces produced by Tasks 1-10。

- [ ] **Step 1: 运行 Go 全量测试**

Run: `cd backend/go && go test ./...`

Expected: PASS，无失败包。

- [ ] **Step 2: 运行前端全量测试**

Run: `cd frontend && pnpm test`

Expected: PASS，无失败测试；不得通过删除断言绕过回归。

- [ ] **Step 3: 运行静态检查与生产构建**

Run:

```bash
cd frontend
pnpm lint
pnpm typecheck
pnpm build
```

Expected: 三条命令退出码均为 0。

- [ ] **Step 4: 扫描 UUID 和旧导航兜底**

Run:

```bash
rg -n 'fallbackAgent|道人 #|Daoist #|session\.agent|s\.agent' frontend
rg -n "path: '/fusion'" frontend/components/layout/nav-config.ts
```

Expected: 第一条没有会话展示命中；第二条只命中金丹阁 child，不命中顶级 nav item。

- [ ] **Step 5: Web 人工验收**

Run: `make dev`

依次验证：

1. 顶栏无独立融合；金丹阁只有“全部金丹 / 融合金丹”，道人府只有“道人列表”。
2. `/fusion` 可访问且金丹阁保持激活。
3. 我的简介可设置 URL 头像、data URI 头像和清空头像；刷新后保持；非法协议被拦截。
4. 丹房论道旧录单聊显示真实道号，群聊显示成员数，页面不出现 UUID。
5. 对谈 Tab 按道人分组且每个道人下可有多个会话；围炉 Tab 只显示群聊。
6. 创建群聊时设置主题成功；留空后首次问答自动命名。
7. 群聊页头可重命名；失败时旧标题不被覆盖、草稿不丢失。

- [ ] **Step 6: Wails 桌面壳验收**

Run: `cd backend/go && wails dev`

如果桌面开发环境要求独立 Python 引擎，先在另一个终端运行 `make dev-python`。重复 Step 5；重点检查窄窗口移动端 Sheet 与桌面侧栏行为一致，单聊深链不依赖道人列表加载。

- [ ] **Step 7: 处理验证结果**

若没有失败，不创建空提交。若任一命令或人工场景失败，停止本任务并回到负责该行为的 Task 1-10：先在该任务列出的测试文件复现，再修改该任务列出的源文件，重新执行该任务的测试与提交步骤；之后从本任务 Step 1 重新开始全量验证。

---

## 完成定义

- 11 个任务的复选框全部完成，并且每个开发任务都有独立提交。
- 所有新增接口字段在 Go DTO、TypeScript 类型、fixture 和 UI 中名称完全一致。
- 任意会话数据缺失场景都不会把 `agent_id` 或其他 UUID 渲染给用户。
- Web 与 Wails 桌面端均完成手工验收。
- `go test ./...`、`pnpm test`、`pnpm lint`、`pnpm typecheck`、`pnpm build` 全部通过。
