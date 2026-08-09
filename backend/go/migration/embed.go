// Package migration 内嵌版本化 SQL 迁移文件(golang-migrate iofs 源)
// 文件命名: {版本号}_{描述}.up.sql / .down.sql
package migration

import "embed"

// FS 内嵌全部迁移 SQL,Docker 镜像无需额外拷贝文件
//
//go:embed *.sql
var FS embed.FS
