package internal

func Init(cfg *Config) {
	// 初始化页面数据管理器
	PageDataMgr = &PageDataManager{
		Debug:    cfg.Server.Mode == "debug",
		PageDir:  cfg.PagesDir,
		Defaults: cfg.PageDefaults,
		Navs:     cfg.PageNavs,
		cacheMap: make(map[string]*PageConfig),
	}
}
