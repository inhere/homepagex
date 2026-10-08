package internal

// LoginInfo 当前登录用户信息
type LoginInfo struct {
	Username    string           `json:"username"`
	Permissions []UserPermission `json:"permissions,omitempty"`
}

// PageDataResponse 页面数据接口响应 DTO
type PageDataResponse struct {
	*PageConfig
	Announcement string `json:"announcement,omitempty"`
	// 覆盖 PageConfig.Navs，仅包含当前用户有权限访问的导航项
	Navs []NavItem `json:"navs"`
	// 当前登录用户信息（游客访问时为 null）
	UserInfo *LoginInfo `json:"user_info,omitempty"`
	// CanWrite 当前身份对当前页面是否可写（前端据此显示/隐藏编辑入口，不再自己算）
	CanWrite bool `json:"can_write"`
	// IconCDNKeys 已配置的图标 CDN key，前端据此拼 icons-local/{key}/... 使用本地缓存
	IconCDNKeys []string `json:"icon_cdn_keys,omitempty"`
	// IconsRemote 服务端是否允许从 CDN 下载图标（配置 icons_remote）。
	// false 时前端不要再去拉 CDN 元数据，否则离线环境会一直卡在加载中。
	IconsRemote bool `json:"icons_remote"`
}
