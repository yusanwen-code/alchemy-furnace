# -*- coding: utf-8 -*-
; ═══════════════════════════════════════════════════════════
; 炼丹炉 · Windows NSIS 安装器(设计 §5 Windows 契约)
;
; 首行 # -*- coding: utf-8 -*- 声明源文件 UTF-8 编码(makensis 按它解码
; 中文, 防乱码);Unicode true 控制产物为 Unicode 安装器, 两者不可互相替代。
;
; 契约点(勿回退, 回归测试见 scripts/tests/verify-windows-package.Tests.ps1
; 与 docs/superpowers/specs/2026-08-28-cross-platform-desktop-release-design.md):
;   1. per-user 安装: InstallDir 固定到 $LOCALAPPDATA, 不写 Program Files
;   2. 不需要管理员权限: RequestExecutionLevel user
;   3. 覆盖升级复用注册表 InstallDir: InstallDirRegKey HKCU(安装时读取旧值)
;   4. 卸载不删除应用数据: 数据目录在 %APPDATA%\AlchemyFurnace
;      (internal/paths os.UserConfigDir), 卸载只删 $INSTDIR 程序文件
;   5. 用户可见名称=炼丹炉(PRODUCT_DISPLAY_NAME), 技术名=AlchemyFurnace
;      (PRODUCT_NAME: 安装目录/注册表/EXE 名保持 ASCII)
; ═══════════════════════════════════════════════════════════
Unicode true

!include "MUI2.nsh"

!ifndef APP_SOURCE
  !error "APP_SOURCE is required"
!endif
!ifndef OUTPUT_FILE
  !error "OUTPUT_FILE is required"
!endif

!define PRODUCT_NAME "AlchemyFurnace"
!define PRODUCT_DISPLAY_NAME "炼丹炉"
!define PRODUCT_EXECUTABLE "AlchemyFurnace.exe"
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

Name "${PRODUCT_DISPLAY_NAME}"
OutFile "${OUTPUT_FILE}"
; 契约: per-user 安装到 LocalAppData, 不要求管理员权限
InstallDir "$LOCALAPPDATA\Programs\${PRODUCT_NAME}"
; 契约: 覆盖升级时复用上次安装目录(升级安装器无需重选路径)
InstallDirRegKey HKCU "Software\${PRODUCT_NAME}" "InstallDir"
RequestExecutionLevel user
ManifestDPIAware true
SetCompressor /SOLID lzma

!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File /r "${APP_SOURCE}\*"

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\${PRODUCT_NAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayName" "${PRODUCT_DISPLAY_NAME}"
  WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  WriteRegStr HKCU "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'

  CreateDirectory "$SMPROGRAMS\${PRODUCT_DISPLAY_NAME}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_DISPLAY_NAME}\${PRODUCT_DISPLAY_NAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  CreateShortcut "$DESKTOP\${PRODUCT_DISPLAY_NAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

Section "Uninstall"
  ; 契约: 只删除程序文件/快捷方式/注册表;应用数据(%APPDATA%\AlchemyFurnace)
  ; 位于 $INSTDIR 之外, 卸载不触碰, 保证用户数据保留
  Delete "$DESKTOP\${PRODUCT_DISPLAY_NAME}.lnk"
  RMDir /r "$SMPROGRAMS\${PRODUCT_DISPLAY_NAME}"
  ; 兼容清理旧英文开始菜单目录(已发布版本的快捷方式残留)
  RMDir /r "$SMPROGRAMS\${PRODUCT_NAME}"
  RMDir /r "$INSTDIR"
  DeleteRegKey HKCU "${UNINSTALL_KEY}"
  DeleteRegKey HKCU "Software\${PRODUCT_NAME}"
SectionEnd
