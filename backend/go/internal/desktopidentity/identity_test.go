// Package desktopidentity 固定「显示名 炼丹炉 / 技术名 AlchemyFurnace」双层命名契约
// 防止常量被错误本地化或改名;消费方见 2026-08-30-desktop-identity-tray-no-console-design.md
package desktopidentity

import "testing"

func TestIdentityContract(t *testing.T) {
	if DisplayName != "炼丹炉" {
		t.Fatalf("DisplayName=%q", DisplayName)
	}
	if TechnicalName != "AlchemyFurnace" {
		t.Fatalf("TechnicalName=%q", TechnicalName)
	}
	if BundleID != "com.alchemyfurnace.desktop" {
		t.Fatalf("BundleID=%q", BundleID)
	}
	if MacBundleName != "炼丹炉.app" {
		t.Fatalf("MacBundleName=%q", MacBundleName)
	}
	if WindowsExecutable != "AlchemyFurnace.exe" {
		t.Fatalf("WindowsExecutable=%q", WindowsExecutable)
	}
}
