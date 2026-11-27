const fs = require('node:fs');
const vm = require('node:vm');

const locale = fs.readFileSync('internal/platform/httpapi/web/locale.js', 'utf8');
const common = fs.readFileSync('internal/platform/httpapi/web/locales/common.js', 'utf8');
const messages = fs.readFileSync('internal/platform/httpapi/web/locales/workspace.js', 'utf8');

module.exports = function workspaceLocale(language = 'zh-CN') {
  const context = vm.createContext({window: {navigator: {language}}});
  vm.runInContext(locale, context);
  vm.runInContext(common, context);
  vm.runInContext(messages, context);
  return context.window.TopbaseLocale;
};
