//go:build darwin

// apply_darwin_test.go - 双 Bundle 名定位(findExtractedApp)
// 新版更新器必须防御性接受 炼丹炉.app 与旧版 AlchemyFurnace.app
package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindExtractedAppAcceptsNewAndLegacyNames(t *testing.T) {
	for _, name := range []string{"炼丹炉.app", "AlchemyFurnace.app"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			want := filepath.Join(root, name)
			require.NoError(t, os.Mkdir(want, 0o755))
			got, err := findExtractedApp(root)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestFindExtractedAppRejectsMissingOrAmbiguousBundle(t *testing.T) {
	root := t.TempDir()
	_, err := findExtractedApp(root)
	require.Error(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, "炼丹炉.app"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "AlchemyFurnace.app"), 0o755))
	_, err = findExtractedApp(root)
	require.Error(t, err)
}

func TestAppBundlePathFromExeRejectsAppTranslocation(t *testing.T) {
	_, err := appBundlePathFromExe("/private/var/folders/ab/cd/AppTranslocation/123/d/炼丹炉.app/Contents/MacOS/炼丹炉")
	require.Error(t, err)
	require.ErrorContains(t, err, "拖入“应用程序”")
}
