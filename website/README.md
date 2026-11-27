# Topbase 官网

官网与在线文档的静态站点源文件。站点直接复用仓库中的 `docs/` Markdown，避免产品文档出现两份内容。

```bash
cd website
npm install
npm run build
npm run preview
```

构建结果位于 `website/dist/`。首页源文件在 `src/`，文档导航与页面由 `scripts/build.mjs` 生成。
