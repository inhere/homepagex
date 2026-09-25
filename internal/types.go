package internal

// LoginInfo 当前登录用户信息
type LoginInfo struct {
	Username    string           `json:"username"`
	Permissions []UserPermission `json:"permissions,omitempty"`
}

// PageDataResponse 页面数据接口响应 DTO
type PageDataResponse struct {
	*PageConfig
	// 覆盖 PageConfig.Navs，仅包含当前用户有权限访问的导航项
	Navs []NavItem `json:"navs"`
	// 当前登录用户信息（游客访问时为 null）
	UserInfo *LoginInfo `json:"user_info,omitempty"`
	// CanWrite 当前身份对当前页面是否可写（前端据此显示/隐藏编辑入口，不再自己算）
	CanWrite bool `json:"can_write"`
	// IconCDNKeys 已配置的图标 CDN key，前端据此拼 icons-local/{key}/... 使用本地缓存
	IconCDNKeys []string `json:"icon_cdn_keys,omitempty"`
}
