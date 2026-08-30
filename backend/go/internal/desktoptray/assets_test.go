package desktoptray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// macOS Template 图标: 22px/44px PNG, 几何与母版一致
func TestMacTemplateIconDimensions(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		size int
	}{
		{"1x", macTemplateIcon, 22},
		{"2x", macTemplateIcon2x, 44},
	} {
		img, err := png.Decode(bytes.NewReader(tc.data))
		require.NoError(t, err)
		require.Equal(t, image.Rect(0, 0, tc.size, tc.size), img.Bounds())
	}
}

// 非透明像素预算: 22px 下 30~300(过少=退化圆点, 过多=实心块)
func TestMacTemplateIconOpacityBudget(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(macTemplateIcon))
	require.NoError(t, err)
	opaque := 0
	for y := 0; y < 22; y++ {
		for x := 0; x < 22; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				opaque++
			}
		}
	}
	require.GreaterOrEqual(t, opaque, 30)
	require.LessOrEqual(t, opaque, 300)
}

// 四角必须透明(Template 图标悬停/按下时才着色)
func TestMacTemplateIconCornersTransparent(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(macTemplateIcon))
	require.NoError(t, err)
	for _, c := range []image.Point{{0, 0}, {21, 0}, {0, 21}, {21, 21}} {
		_, _, _, a := img.At(c.X, c.Y).RGBA()
		require.Zero(t, a, "四角 (%d,%d) 必须透明", c.X, c.Y)
	}
}

// Windows ICO: 目录必须含 16/20/24/32/48 五种帧(设计 §6 Windows 托盘)
func TestWindowsIconFrameSizes(t *testing.T) {
	require.GreaterOrEqual(t, len(windowsIcon), 6+16, "ICO 至少要有 header + 一帧")
	var count uint16
	require.NoError(t, binary.Read(bytes.NewReader(windowsIcon[4:6]), binary.LittleEndian, &count))
	require.GreaterOrEqual(t, int(count), 5)

	sizes := map[byte]bool{}
	for i := 0; i < int(count); i++ {
		entry := windowsIcon[6+i*16:]
		sizes[entry[0]] = true
		sizes[entry[1]] = true
	}
	for _, want := range []byte{16, 20, 24, 32, 48} {
		require.True(t, sizes[want], "ICO 缺少 %dpx 帧", want)
	}
}
