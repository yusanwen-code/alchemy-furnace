# ═══════════════════════════════════════════════════════════
# 炼丹炉 · verify-windows-package.ps1 的 Pester v5 测试
#
# 在 Windows x64 runner 运行(pwsh + Pester v5):
#   pwsh -NoProfile -Command "Install-Module Pester -Force -Scope CurrentUser -SkipPublisherCheck; Invoke-Pester scripts/tests/verify-windows-package.Tests.ps1"
#
# 覆盖: PE 架构解析 / 错误架构 / 缺 runtime / import 失败 /
#       空安装器 / 版本不一致 / 全绿通过
# ═══════════════════════════════════════════════════════════

BeforeAll {
  $script:Verifier = (Resolve-Path (Join-Path $PSScriptRoot '..\verify-windows-package.ps1')).Path
  $script:FixtureRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('af-verify-windows-' + [System.Guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force -Path $script:FixtureRoot | Out-Null

  # 生成最小合法 PE(DOS + PE 头),Machine 可指定,可附加 payload(用于版本字节搜索)
  function New-FakePe {
    param([string]$Path, [uint16]$Machine, [byte[]]$Payload = @())
    $fs = [System.IO.File]::Create($Path)
    try {
      $bw = New-Object System.IO.BinaryWriter($fs)
      $bw.Write([byte]0x4D); $bw.Write([byte]0x5A)   # 'MZ'
      $bw.BaseStream.Position = 0x3C
      $bw.Write([uint32]0x80)                        # e_lfanew = 0x80
      $bw.BaseStream.Position = 0x80
      $bw.Write([uint32]0x00004550)                  # 'PE\0\0'
      $bw.Write([uint16]$Machine)                    # COFF Machine
      $bw.Write($Payload)
      $bw.Flush()
    } finally { $fs.Dispose() }
  }

  # 构造与 ci-assemble.sh 一致的包布局:
  #   <root>/AlchemyFurnace.exe
  #   <root>/runtime/python.exe
  #   <root>/runtime/engine/app/main.py
  function New-PackageFixture {
    param(
      [string]$Root,
      [bool]$WithRuntime = $true,
      [uint16]$AppMachine = 0x8664,
      [uint16]$PythonMachine = 0x8664,
      [byte[]]$AppPayload = ([System.Text.Encoding]::ASCII.GetBytes('v0.1.1'))
    )
    New-FakePe -Path (Join-Path $Root 'AlchemyFurnace.exe') -Machine $AppMachine -Payload $AppPayload
    if ($WithRuntime) {
      $engineApp = Join-Path $Root 'runtime\engine\app'
      New-Item -ItemType Directory -Force -Path $engineApp | Out-Null
      New-FakePe -Path (Join-Path $Root 'runtime\python.exe') -Machine $PythonMachine
      Set-Content -Path (Join-Path $engineApp 'main.py') -Value '"""fixture 引擎"""' -Encoding UTF8
    }
  }

  # dot-source 脚本以暴露内部函数(脚本主入口有 dot-source 保护,不会自动执行)
  . $script:Verifier

  # 命名契约: 指向真实 installer.nsi(UTF-8 显式编码 + 中文显示名)
  $script:InstallerScript = (Resolve-Path (Join-Path $PSScriptRoot '..\..\backend\go\build\windows\installer.nsi')).Path
}

AfterAll {
  Remove-Item -Recurse -Force $script:FixtureRoot -ErrorAction SilentlyContinue
}

Describe 'verify-windows-package.ps1 架构解析' {
  It 'Get-PeMachine 正确解析 x64(0x8664)与 x86(0x014C) PE 文件' {
    $x64 = Join-Path $script:FixtureRoot 'unit-x64.exe'
    $x86 = Join-Path $script:FixtureRoot 'unit-x86.exe'
    New-FakePe -Path $x64 -Machine 0x8664
    New-FakePe -Path $x86 -Machine 0x014C
    Get-PeMachine $x64 | Should -Be 0x8664
    Get-PeMachine $x86 | Should -Be 0x014C
  }
}

Describe 'verify-windows-package.ps1 负例' {
  It '错误架构: 应用为 32 位 PE(0x014C)时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'arch-app-x86'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg -AppMachine 0x014C
    $inst = Join-Path $script:FixtureRoot 'arch-app-x86-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match '应用架构错误'
  }

  It '错误架构: 内嵌 python.exe 为 32 位 PE(0x014C)时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'arch-py-x86'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg -PythonMachine 0x014C
    $inst = Join-Path $script:FixtureRoot 'arch-py-x86-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match 'python.exe 架构错误'
  }

  It '缺 runtime: 没有 python.exe 时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'no-runtime'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg -WithRuntime $false
    $inst = Join-Path $script:FixtureRoot 'no-runtime-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match '缺少 python.exe'
  }

  It 'import 失败: python.exe 无法执行/契约检查失败时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'py-not-runnable'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg   # python.exe 为最小假 PE: 机器码正确但不可执行
    $inst = Join-Path $script:FixtureRoot 'py-not-runnable-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match '契约检查失败'
  }

  It '空安装器: 0 字节安装器时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'empty-installer'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg
    $inst = Join-Path $script:FixtureRoot 'empty-setup.exe'
    New-Item -ItemType File -Path $inst | Out-Null   # 0 字节
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match '安装器为空'
  }

  It '版本不一致: 应用内嵌版本与 ExpectedVersion 不同时验证失败' {
    $pkg = Join-Path $script:FixtureRoot 'version-mismatch'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg -AppPayload ([System.Text.Encoding]::ASCII.GetBytes('v0.1.0'))
    $inst = Join-Path $script:FixtureRoot 'version-mismatch-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    $out = (& $script:Verifier -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1' 2>&1 6>&1 | Out-String)
    $LASTEXITCODE | Should -Be 1
    $out | Should -Match '版本不一致'
  }
}

Describe 'verify-windows-package.ps1 命名契约(双层命名)' {
  It '真实 installer.nsi 必须显式 UTF-8 编码 + 中文显示名 + 中文开始菜单目录' {
    Test-DesktopDisplayNameContract -InstallerScript $script:InstallerScript | Should -BeTrue
  }

  It '缺少 UTF-8 编码声明的 NSIS 文件必须被拒绝' {
    $bad = Join-Path $script:FixtureRoot 'bad.nsi'
    Set-Content -Path $bad -Value '; no encoding declaration' -Encoding ASCII
    Test-DesktopDisplayNameContract -InstallerScript $bad | Should -BeFalse
  }
}

Describe 'verify-windows-package.ps1 全绿通过' {
  It '完整 fixture 全部检查通过(退出码 0)' {
    $pkg = Join-Path $script:FixtureRoot 'happy'
    New-Item -ItemType Directory -Force -Path $pkg | Out-Null
    New-PackageFixture -Root $pkg
    $inst = Join-Path $script:FixtureRoot 'happy-setup.exe'
    Set-Content -Path $inst -Value 'installer' -Encoding ASCII
    # 假 PE 无法真实执行,这里 mock 掉 python 契约(真实 import 由 CI 真包验证)
    Mock Invoke-PythonContractCheck { return [pscustomobject]@{ Success = $true; Output = 'ARCH=AMD64 UTF8=1 IMPORT=ok'; ExitCode = 0 } }
    $rc = Invoke-VerifyWindowsPackage -PackageDir $pkg -Installer $inst -ExpectedVersion 'v0.1.1'
    $rc | Should -Be 0
  }
}
