# Topbase website and documentation

The homepage templates live in `src/`. The published user guide is built from paired chapters under `guides/en/` and `guides/zh/`. Edit both languages together. `scripts/build.mjs` defines the chapter order and checks that each pair references the same screenshot steps and includes the referenced sample files.

Use Node.js 22 or later. From the repository root, run:

```bash
cd website
npm ci
npm run build
npm run preview
```

Open `http://localhost:4173/` for English or `http://localhost:4173/zh/` for Chinese. The static output in `dist/` is published to GitHub Pages.

Capture guide screenshots from a clean checkout of the release documented by the guide. Start an isolated instance and check `/api/version` before signing in. Use the downloadable sample orders and the appropriate UI language, perform the documented action, and capture the actual application at 2× device scale. Check the button labels, results and image dimensions in the built page. Store paired captures in `src/assets/guide-images/`.

## 中文说明

首页模板位于 `src/`，在线手册正文位于 `guides/en/` 和 `guides/zh/`。请同步更新同一章的两个语言版本。`scripts/build.mjs` 管理章节顺序，并检查中英文截图步骤及示例文件。

使用 Node.js 22 或更高版本，从仓库根目录执行上方命令。打开 `http://localhost:4173/zh/` 预览中文站点，英文首页位于 `http://localhost:4173/`。GitHub Pages 发布 `dist/` 中的静态文件。

教程截图使用文档所对应发行版的干净检出。启动隔离实例，先通过 `/api/version` 核对版本，再使用可下载的模拟订单和对应的界面语言登录，实际完成所写操作，以 2 倍像素密度截图。在构建出的页面中验证按钮名称、数据结果和图片尺寸。中英文配对截图存放在 `src/assets/guide-images/`。
