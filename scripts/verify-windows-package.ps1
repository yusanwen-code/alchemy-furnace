#!/usr/bin/env pwsh
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · Windows x64 包校验器
#
# 校验打包产物是否满足 windows-amd64 契约(设计 §5 Windows + §4 运行时):
#   - AlchemyFurnace.exe 与内嵌 runtime/python.exe 的 PE Machine 必须为 0x8664(x64)
#   - python.exe 报告 AMD64 + UTF-8 mode=1, 且能从 engine/ 导入 app.main
#   - runtime/engine/app/main.py 存在且非空
#   - 安装器(Setup.exe)存在且非空
#   - 应用内嵌版本与 ExpectedVersion 一致(ldflags -X buildinfo.Version 注入的字节)
#
# 用法(pwsh 7 或 Windows PowerShell 5.1 均兼容):
#   pwsh scripts/verify-windows-package.ps1 `
#     -PackageDir backend/go/build/package/windows `
#     -Installer backend/go/build/dist/AlchemyFurnace-Setup.exe `
#     -ExpectedVersion v0.1.1 `
#     [-LaunchSmoke]      # 真实启动烟测: ALCHEMY_SMOKE=1 + 临时数据目录,
#                         # 确认主进程存活 + python.exe 子进程 + 无窗口句柄(无黑框)
#
# 退出码: 0=通过 1=校验失败 2=参数错误
# 测试: scripts/tests/verify-windows-package.Tests.ps1(Pester v5, Windows runner)
# ═══════════════════════════════════════════════════════════

[CmdletBinding()]
param(
  [string]$PackageDir = '',
  [string]$Installer = '',
  [string]$ExpectedVersion = '',
  [switch]$LaunchSmoke
)

# ─── 内部函数(允许 Pester dot-source 后单测) ───

# 规范化路径(相对路径按调用方 CWD 解析, 不做存在性要求)
function Get-AbsolutePath {
  param([string]$Path)
  if ([string]::IsNullOrWhiteSpace($Path)) { return '' }
  return [System.IO.Path]::GetFullPath($Path)
}

# 命名契约(双层命名): 安装器源文件必须显式 UTF-8 编码 + 中文显示名 + 中文开始菜单目录
# 返回布尔; 由 Pester 单测与真实安装验收共同守护
function Test-DesktopDisplayNameContract {
  param([string]$InstallerScript)
  $text = [IO.File]::ReadAllText($InstallerScript, [Text.Encoding]::UTF8)
  return $text.StartsWith("# -*- coding: utf-8 -*-") -and
    $text.Contains('!define PRODUCT_DISPLAY_NAME "炼丹炉"') -and
    $text.Contains('$SMPROGRAMS\${PRODUCT_DISPLAY_NAME}')
}

# 真实安装后的用户可见名称验收(Windows 人工/自动验收环境):
# 注册表 Uninstall DisplayName、开始菜单中文目录快捷方式、桌面快捷方式,
# 三处必须逐字符等于"炼丹炉"。返回 [pscustomobject]@{ All; RegName; StartMenu; Desktop }
function Test-InstalledDisplayNames {
  param()
  $okReg = $okStart = $okDesktop = $false
  $regName = ''
  try {
    $key = Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\AlchemyFurnace' -ErrorAction Stop
    $regName = [string]$key.DisplayName
    $okReg = ($regName -eq '炼丹炉')
  } catch { $regName = '' }

  $startLnk = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\炼丹炉\炼丹炉.lnk'
  $okStart = (Test-Path -LiteralPath $startLnk)
  $desktopLnk = Join-Path ([Environment]::GetFolderPath('Desktop')) '炼丹炉.lnk'
  $okDesktop = (Test-Path -LiteralPath $desktopLnk)

  $all = $okReg -and $okStart -and $okDesktop
  return [pscustomobject]@{ All = $all; RegName = $regName; StartMenu = $okStart; Desktop = $okDesktop }
}

# 读取 PE 头 Machine 字段: 0x8664=AMD64(x64), 0x014C=I386(32 位), 0xAA64=ARM64
function Get-PeMachine {
  param([string]$FilePath)
  $fs = [System.IO.File]::OpenRead($FilePath)
  try {
    $br = New-Object System.IO.BinaryReader($fs)
    $fs.Position = 0x3C
    $peOffset = $br.ReadUInt32()
    $fs.Position = $peOffset
    $sig = $br.ReadUInt32()
    if ($sig -ne 0x00004550) {
      throw "不是有效的 PE 文件(签名 0x$('{0:X8}' -f $sig)): $FilePath"
    }
    return $br.ReadUInt16()
  } finally {
    $fs.Dispose()
  }
}

# 读取 PE Optional Header Subsystem 字段: 2=Windows GUI, 3=Console
# Optional Header 起点 = PE 头 + COFF 头(24 字节); Subsystem 在 PE32+/PE32 的 +0x44
function Get-PeSubsystem {
  param([Parameter(Mandatory)][string]$Path)
  $stream = [IO.File]::OpenRead((Get-AbsolutePath $Path))
  $reader = [IO.BinaryReader]::new($stream)
  try {
    $stream.Position = 0x3C
    $peOffset = $reader.ReadInt32()
    $stream.Position = $peOffset
    if ($reader.ReadUInt32() -ne 0x00004550) { throw "PE signature invalid" }
    $stream.Position = $peOffset + 24
    $magic = $reader.ReadUInt16()
    if ($magic -ne 0x10B -and $magic -ne 0x20B) { throw "optional header magic invalid" }
    $stream.Position = $peOffset + 24 + 0x44
    return $reader.ReadUInt16()
  } finally {
    $reader.Dispose()
    $stream.Dispose()
  }
}

# 版本一致: 期望版本(ldflags -X 注入)必须以 ASCII 字节出现在应用二进制中
# 返回字符串: 'match'=一致 / 'mismatch'=不一致 / 'skip'=非 SemVer(如 dev)不做字节校验
# 注意: 不用布尔值返回, 避免 PowerShell 把 'skip' 强制转布尔导致误判
function Test-VersionEmbedded {
  param([string]$FilePath, [string]$ExpectedVersion)
  $needle = $ExpectedVersion.TrimStart('v')
  if ($needle -notmatch '^\d+\.\d+\.\d+') {
    return 'skip'
  }
  try {
    $hay = [System.Text.Encoding]::Latin1.GetString([System.IO.File]::ReadAllBytes($FilePath))
    $n1 = [System.Text.Encoding]::Latin1.GetString([System.Text.Encoding]::ASCII.GetBytes('v' + $needle))
    $n2 = [System.Text.Encoding]::Latin1.GetString([System.Text.Encoding]::ASCII.GetBytes($needle))
    if ($hay.IndexOf($n1, [System.StringComparison]::Ordinal) -ge 0) { return 'match' }
    if ($hay.IndexOf($n2, [System.StringComparison]::Ordinal) -ge 0) { return 'match' }
    return 'mismatch'
  } catch {
    return 'mismatch'
  }
}

# python 契约: 从 engine/ 运行内嵌解释器, 必须报告 AMD64 + UTF-8 mode=1 + import app.main
# 返回: [pscustomobject]@{ Success; Output; ExitCode }
function Invoke-PythonContractCheck {
  param([string]$PythonExe, [string]$EngineDir)
  # Push-Location 切换目录后相对路径会失效, 先绝对化(计划 Task 1 的路径教训)
  $PythonExe = Get-AbsolutePath $PythonExe
  $EngineDir = Get-AbsolutePath $EngineDir
  $code = 'import platform, sys; import app.main; print("ARCH=" + platform.machine() + " UTF8=" + str(sys.flags.utf8_mode) + " IMPORT=ok")'
  $out = ''
  $started = $false
  $exitCode = 0
  $pushed = $false
  try {
    Push-Location $EngineDir
    $pushed = $true
    $oldUtf8 = $env:PYTHONUTF8
    $oldIo = $env:PYTHONIOENCODING
    $env:PYTHONUTF8 = '1'
    $env:PYTHONIOENCODING = 'utf-8'
    try {
      $raw = & $PythonExe -c $code 2>&1
      $started = $?
      # $LASTEXITCODE 只在原生命令真正执行后才会创建; 未执行时读取在 StrictMode 下会抛错
      if ($started -and (Test-Path variable:LASTEXITCODE)) { $exitCode = $LASTEXITCODE }
      $out = ($raw | Out-String)
    } catch {
      $started = $false
      $out = $_.Exception.Message
    } finally {
      $env:PYTHONUTF8 = $oldUtf8
      $env:PYTHONIOENCODING = $oldIo
    }
  } catch {
    $started = $false
    $out = $_.Exception.Message
  } finally {
    if ($pushed) { Pop-Location }
  }
  $ok = $started -and ($exitCode -eq 0) -and
    $out -match 'AMD64' -and $out -match 'UTF8=1' -and $out -match 'IMPORT=ok'
  return [pscustomobject]@{ Success = $ok; Output = $out; ExitCode = $exitCode }
}

# ─── 启动烟测(仅 -LaunchSmoke 启用) ───
# ALCHEMY_SMOKE=1 启动主 exe(只起 HTTP 不开窗),AppData 重定向到临时目录避免
# 触碰真实用户数据;等待进程存活并出现 python.exe 直接子进程;主进程与 python
# 的 MainWindowHandle 必须均为 0(任何非零句柄=黑框/窗口泄漏);Stop-Process 清理。
# 返回: [string] '' = 通过 / 错误描述
function Invoke-LaunchSmoke {
  param([Parameter(Mandatory)][string]$PackageDir)
  $exe = Join-Path (Get-AbsolutePath $PackageDir) 'AlchemyFurnace.exe'
  if (-not (Test-Path -LiteralPath $exe)) {
    return "烟测失败: 未找到主程序 $exe"
  }
  $smokeHome = Join-Path ([System.IO.Path]::GetTempPath()) ('af-smoke-' + [Guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force -Path $smokeHome | Out-Null
  $oldAppData = $env:AppData
  $oldSmoke = $env:ALCHEMY_SMOKE
  $proc = $null
  $py = $null
  try {
    # PS 5.1 无 Start-Process -Environment,用环境变量继承传递
    $env:AppData = $smokeHome
    $env:ALCHEMY_SMOKE = '1'
    $proc = Start-Process -FilePath $exe -PassThru

    # 等待: 主进程存活 且 出现 python.exe 直接子进程(最多 20s)
    $deadline = (Get-Date).AddSeconds(20)
    while ($true) {
      $p = Get-Process -Id $proc.Id -ErrorAction SilentlyContinue
      if (-not $p -or $p.HasExited) {
        return "烟测失败: 主进程提前退出(数据目录=$smokeHome)"
      }
      $children = @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$($proc.Id)" -ErrorAction SilentlyContinue)
      $py = $children | Where-Object { $_.Name -ieq 'python.exe' } | Select-Object -First 1
      if ($py) { break }
      if ((Get-Date) -gt $deadline) {
        return '烟测失败: 20s 内未发现 python.exe 子进程'
      }
      Start-Sleep -Milliseconds 500
    }

    # 无窗口句柄契约: smoke 模式主进程不开主窗;任何非零句柄=出现黑框/console
    $hMain = (Get-Process -Id $proc.Id -ErrorAction SilentlyContinue).MainWindowHandle
    $hPy = (Get-Process -Id $py.ProcessId -ErrorAction SilentlyContinue).MainWindowHandle
    if ($null -ne $hMain -and $hMain -ne 0) {
      return "烟测失败: 主进程出现窗口句柄 $hMain (PID $($proc.Id))"
    }
    if ($null -ne $hPy -and $hPy -ne 0) {
      return "烟测失败: python 出现窗口句柄 $hPy (PID $($py.ProcessId))"
    }
    return ''
  } finally {
    if ($null -ne $proc) { Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue }
    if ($null -ne $py) { Stop-Process -Id $py.ProcessId -Force -ErrorAction SilentlyContinue }
    $env:AppData = $oldAppData
    $env:ALCHEMY_SMOKE = $oldSmoke
    Remove-Item -Recurse -Force $smokeHome -ErrorAction SilentlyContinue
  }
}

# 主校验流程, 返回退出码(0 通过 / 1 失败)
function Invoke-VerifyWindowsPackage {
  param(
    [string]$PackageDir,
    [string]$Installer,
    [string]$ExpectedVersion,
    [switch]$LaunchSmoke
  )
  $failures = [System.Collections.Generic.List[string]]::new()
  $pkg = Get-AbsolutePath $PackageDir
  $app = Join-Path $pkg 'AlchemyFurnace.exe'
  $runtimeDir = Join-Path $pkg 'runtime'
  $pythonExe = Join-Path $runtimeDir 'python.exe'
  $engineDir = Join-Path $runtimeDir 'engine'
  $mainPy = Join-Path (Join-Path $engineDir 'app') 'main.py'

  Write-Host "[verify-windows] 校验 Windows x64 包: $pkg (期望版本 $ExpectedVersion)"

  # 1. 应用: 存在 + PE Machine 0x8664 + Subsystem 2 (Windows GUI)
  if (-not (Test-Path -LiteralPath $app)) {
    $failures.Add("未找到应用: $app")
  } elseif ((Get-Item -LiteralPath $app).Length -eq 0) {
    $failures.Add("应用为空文件: $app")
  } else {
    try { $appMachine = Get-PeMachine $app } catch { $appMachine = 0 }
    if ($appMachine -ne 0x8664) {
      $failures.Add(('应用架构错误(期望 0x8664, 实际 0x{0:X4}): {1}' -f $appMachine, $app))
    } else {
      Write-Host "[verify-windows] OK 应用架构 x64 (0x8664): $app"
    }
    # GUI Subsystem 契约: 主 exe 必须是 2(Windows GUI), 否则启动闪黑框
    # (runtime/python.exe 允许 Console(3), 由启动参数 CREATE_NO_WINDOW 隐藏)
    try { $subsystem = Get-PeSubsystem $app } catch { $subsystem = -1 }
    if ($subsystem -ne 2) {
      $failures.Add("应用 PE Subsystem=$subsystem，期望 2 (Windows GUI): $app")
    } else {
      Write-Host "[verify-windows] OK Windows GUI subsystem (2): $app"
    }
  }

  # 2. runtime: python.exe 存在 + Machine 0x8664; engine/app/main.py 存在非空; 契约检查
  if (-not (Test-Path -LiteralPath $pythonExe)) {
    $failures.Add("运行时缺少 python.exe: $pythonExe")
  } else {
    try { $pyMachine = Get-PeMachine $pythonExe } catch { $pyMachine = 0 }
    if ($pyMachine -ne 0x8664) {
      $failures.Add(('python.exe 架构错误(期望 0x8664, 实际 0x{0:X4}): {1}' -f $pyMachine, $pythonExe))
    } else {
      Write-Host "[verify-windows] OK python.exe 架构 x64 (0x8664): $pythonExe"
      if (-not (Test-Path -LiteralPath $mainPy)) {
        $failures.Add("engine/app/main.py 缺失: $mainPy")
      } elseif ((Get-Item -LiteralPath $mainPy).Length -eq 0) {
        $failures.Add("engine/app/main.py 为空: $mainPy")
      } else {
        Write-Host "[verify-windows] OK engine/app/main.py 存在: $mainPy"
        $pc = Invoke-PythonContractCheck -PythonExe $pythonExe -EngineDir $engineDir
        if (-not $pc.Success) {
          $detail = ($pc.Output -replace '(?m)^', '      ')
          $failures.Add(("python.exe 契约检查失败(需 AMD64 + UTF-8 mode=1 + import app.main): $pythonExe`n$detail"))
        } else {
          Write-Host "[verify-windows] OK python 契约: $($pc.Output.Trim())"
        }
      }
    }
  }

  # 3. 安装器非空
  if (-not (Test-Path -LiteralPath $Installer)) {
    $failures.Add("安装器不存在: $Installer")
  } elseif ((Get-Item -LiteralPath $Installer).Length -eq 0) {
    $failures.Add("安装器为空: $Installer")
  } else {
    $size = (Get-Item -LiteralPath $Installer).Length
    Write-Host "[verify-windows] OK 安装器非空 ($size bytes): $Installer"
  }

  # 4. 版本一致(应用内嵌 buildinfo.Version)
  $ver = Test-VersionEmbedded -FilePath $app -ExpectedVersion $ExpectedVersion
  if ($ver -eq 'skip') {
    Write-Host "[verify-windows] 跳过版本字节校验(非 SemVer: $ExpectedVersion)"
  } elseif ($ver -eq 'mismatch') {
    $failures.Add("版本不一致(期望 $ExpectedVersion): $app")
  } else {
    Write-Host "[verify-windows] OK 版本一致 ($ExpectedVersion): $app"
  }

  # 5. 可选启动烟测: ALCHEMY_SMOKE=1 真实拉起, 验证无黑框(默认关闭, release 启用)
  if ($LaunchSmoke) {
    $smokeErr = Invoke-LaunchSmoke -PackageDir $pkg
    if ($smokeErr) {
      $failures.Add($smokeErr)
    } else {
      Write-Host '[verify-windows] OK 启动烟测: 主进程存活 + python.exe 子进程 + 无窗口句柄'
    }
  }

  if ($failures.Count -gt 0) {
    Write-Host '[verify-windows] ❌ Windows x64 包验证失败:'
    foreach ($f in $failures) { Write-Host "  - $f" }
    return 1
  }
  Write-Host '[verify-windows] ✅ Windows x64 包验证通过'
  return 0
}

# ─── 主入口(dot-source 时跳过, 供 Pester 直接调用内部函数) ───
if ($MyInvocation.InvocationName -ne '.') {
  Set-StrictMode -Version Latest
  $ErrorActionPreference = 'Stop'
  if (-not $PackageDir -or -not $Installer -or -not $ExpectedVersion) {
    Write-Host '用法: verify-windows-package.ps1 -PackageDir <dir> -Installer <setup.exe> -ExpectedVersion <vX.Y.Z> [-LaunchSmoke]'
    exit 2
  }
  exit (Invoke-VerifyWindowsPackage -PackageDir $PackageDir -Installer $Installer -ExpectedVersion $ExpectedVersion -LaunchSmoke:$LaunchSmoke)
}
