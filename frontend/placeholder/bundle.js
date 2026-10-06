// 占位脚本：仅用于「未带 -tags embedfrontend 编译」时前端资源目录结构完整。
// 真正的入口脚本是 `pnpm --dir frontend run build` 生成的 build/bundle.js。
console.warn(
  '[homepagex] 前端资源未内嵌（占位 bundle.js）。请用 `make build` 重新编译，' +
    '或在配置里把 frontend_dir 指向 pnpm build 的产物目录。'
);
