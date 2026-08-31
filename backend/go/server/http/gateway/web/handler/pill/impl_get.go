// 旧金丹详情入口（任务 5 起改道）
// 旧 GET /api/v1/pills/:uuid 仅提供 LegacyMap 跳转 {entity_type:"recipe", recipe_id:"..."}，
// 该路由已指向 pill_inventory.Handler.ResolveLegacyPill（见 router.go）；
// 本文件原 Get/parseUUID 不再被路由引用，避免旧数据从旧入口泄漏（plan 任务 5 行 431）。
package pill
