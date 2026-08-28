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
#     -ExpectedVersion v0.1.1
#
# 退出码: 0=通过 1=校验失败 2=参数错误
# 测试: scripts/tests/verify-windows-package.Tests.ps1(Pester v5, Windows runner)
# ═══════════════════════════════════════════════════════════

[CmdletBinding()]
param(
  [string]$PackageDir = '',
  [string]$Installer = '',
  [string]$ExpectedVersion = ''
)

# ─── 内部函数(允许 Pester dot-source 后单测) ───

# 规范化路径(相对路径按调用方 CWD 解析, 不做存在性要求)
function Get-AbsolutePath {
  param([string]$Path)
  if ([string]::IsNullOrWhiteSpace($Path)) { return '' }
  return [System.IO.Path]::GetFullPath($Path)
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

# 主校验流程, 返回退出码(0 通过 / 1 失败)
function Invoke-VerifyWindowsPackage {
  param(
    [string]$PackageDir,
    [string]$Installer,
    [string]$ExpectedVersion
  )
  $failures = [System.Collections.Generic.List[string]]::new()
  $pkg = Get-AbsolutePath $PackageDir
  $app = Join-Path $pkg 'AlchemyFurnace.exe'
  $runtimeDir = Join-Path $pkg 'runtime'
  $pythonExe = Join-Path $runtimeDir 'python.exe'
  $engineDir = Join-Path $runtimeDir 'engine'
  $mainPy = Join-Path (Join-Path $engineDir 'app') 'main.py'

  Write-Host "[verify-windows] 校验 Windows x64 包: $pkg (期望版本 $ExpectedVersion)"

  # 1. 应用: 存在 + PE Machine 0x8664
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
    Write-Host '用法: verify-windows-package.ps1 -PackageDir <dir> -Installer <setup.exe> -ExpectedVersion <vX.Y.Z>'
    exit 2
  }
  exit (Invoke-VerifyWindowsPackage -PackageDir $PackageDir -Installer $Installer -ExpectedVersion $ExpectedVersion)
}
