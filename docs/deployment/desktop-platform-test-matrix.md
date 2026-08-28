# 桌面平台测试矩阵（安装与数据保留验收）

> 本文档是 `docs/deployment/desktop-release.md` 的配套验收记录，按发布计划的 Task 7
> 验收清单逐项建表。Release 门禁：**任一平台未验收，RC 不得晋升 stable。**

## 1. 范围与资产

支持的发布平台（与设计文档 §1 一致）：

| 平台键 | 系统范围 | CPU | 人工安装资产 | 自动更新资产 |
|---|---|---|---|---|
| darwin-arm64 | macOS 12+ | Apple Silicon | AlchemyFurnace-mac-arm64.dmg | AlchemyFurnace-mac-arm64.zip |
| darwin-amd64 | macOS 12+ | Intel x86_64 | AlchemyFurnace-mac-x64.dmg | AlchemyFurnace-mac-x64.zip |
| windows-amd64 | Windows 10/11 x64 | x86_64 | AlchemyFurnace-Setup.exe | 同安装器 |

每版 stable Release 必须包含 5 个平台二进制、`checksums.txt`、`release-manifest.json`
（完整性由 `scripts/verify-release-assets.sh` 与 draft 远端复验把关，见
`docs/deployment/desktop-release.md`）。本矩阵只验收「目标系统上的真实安装与使用」。

## 2. 平台与资产登记

验收前逐平台登记测试环境；SHA256 取自 Release 资产的 `checksums.txt`，与本地
`shasum -a 256 <资产>` 复核一致后再开始验收。

| 平台 | 测试环境（真机/VM） | OS build | CPU | 资产名 | SHA256 | 截图目录 | 日志目录 |
|---|---|---|---|---|---|---|---|
| ARM Mac |  |  |  | AlchemyFurnace-mac-arm64.dmg / .zip |  |  |  |
| Intel Mac |  |  |  | AlchemyFurnace-mac-x64.dmg / .zip |  |  |  |
| Windows 10 x64 |  |  |  | AlchemyFurnace-Setup.exe |  |  |  |
| Windows 11 x64 |  |  |  | AlchemyFurnace-Setup.exe |  |  |  |

## 3. 验收矩阵

单元格格式：`结果(日期) · 截图: <路径> · 日志: <路径>`。留空或 `未验收` 即视为未通过。

| 验收项 | ARM Mac (macOS 12+) | Intel Mac (macOS 12+) | Windows 10 x64 | Windows 11 x64 |
|---|---|---|---|---|
| 安装（DMG 拖入 Applications / Setup.exe 完成安装） |  |  |  |  |
| 启动（冷启动首屏正常，含 302 到 http origin 的 webview 加载） |  |  |  |  |
| 原生架构（Intel Mac 确认非 Rosetta 下运行 ARM 程序；ARM Mac 确认 arm64） |  |  |  |  |
| Engine readiness（Python 引擎自检/健康，`import app.main` 成功、UTF-8 模式开启） |  |  |  |  |
| 首页（静态资源、API 通、创建道人/金丹/会话可用） |  |  |  |  |
| Skill 导出（金丹导出落盘到数据目录，可打开所在文件夹） |  |  |  |  |
| 重开（退出后重新打开，会话/设置状态保留） |  |  |  |  |
| ZIP 更新（更新器选择对应架构 ZIP/Setup.exe 并成功安装） |  |  |  |  |
| WebView2 缺失（缺 WebView2 时启动内嵌 bootstrapper，给出可执行安装流程与明确错误） | 不适用 | 不适用 |  |  |
| 快捷方式（桌面/开始菜单创建） | 不适用 | 不适用 |  |  |
| 卸载（卸载不删除应用数据目录） | 不适用 | 不适用 |  |  |
| 升级数据保留（升级前创建道人/金丹/会话，升级后全部保留） |  |  |  |  |
| 断网行为（断网本地功能可用；联网调用显示可理解错误） |  |  |  |  |

## 4. 专项验收步骤

### 4.1 升级数据保留（全部平台）

1. 安装上一正式版本（v0.1.0 或更高），启动并完成一次 Python engine readiness。
2. 创建至少 1 个道人、1 个金丹、1 个会话（含一条消息）。
3. 关闭应用，安装目标版本（Mac：DMG 覆盖安装或 ZIP 更新；Windows：Setup.exe 覆盖升级）。
4. 重新启动，确认数据目录（macOS `~/Library/Application Support/AlchemyFurnace/`；
   Windows `%LOCALAPPDATA%\AlchemyFurnace\`）中 SQLite 与 `secret.key` 未重建、未丢失。
5. 确认步骤 2 创建的道人/金丹/会话完整可读。

### 4.2 断网行为（全部平台）

1. 断网（或防火墙拦截出站）后冷启动应用：本地功能（首页、本地 API 健康检查、
   已有会话浏览）必须可用。
2. 发起一次依赖网络的调用（如模型请求）：必须显示可理解的错误信息，不崩溃、
   不白屏，恢复联网后调用恢复正常。

### 4.3 Intel Mac 原生架构确认

启动应用后用终端核对（在应用外执行）：

```bash
arch -x86_64 true   # 确保 shell 是 x86_64
file /Applications/AlchemyFurnace.app/Contents/MacOS/AlchemyFurnace
# 期望: ... x86_64 而非 "universal" 或 "arm64"
```

若显示 arm64，说明当前是 Rosetta 下运行的 ARM 程序，该行判 FAIL。

## 5. 验收结论

| 平台 | 结论（通过/未通过） | 验收人 | 日期 | 备注 |
|---|---|---|---|---|
| ARM Mac |  |  |  |  |
| Intel Mac |  |  |  |  |
| Windows 10 x64 |  |  |  |  |
| Windows 11 x64 |  |  |  |  |

**门禁：任一平台未验收（未通过或未执行），RC 不得晋升 stable。** 验收全部通过后，
由维护者按 `docs/deployment/desktop-release.md` 的发布流程从同一 commit 发布
v0.1.1（或更高），并同步三平台分别检查更新确认选择正确资产。

## 6. 维护说明

- 本矩阵由发布负责人维护；每版 RC 验收前清空「验收矩阵」「验收结论」再逐项填写。
- 截图与日志按平台目录归档（路径填入「平台与资产登记」表），命名含日期与验收项，
  如 `2026-08-28-intel-install.png`。
- Windows 侧 PowerShell/Pester 相关测试项须在 Windows 真机/VM 执行，macOS 本机
  无法替代；同样地，不用 ARM Mac 假装 Intel 验收，不用 Wine 替代 Windows 10/11。
