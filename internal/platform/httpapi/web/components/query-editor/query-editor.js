(function (global) {
  'use strict';

  function resolve(target) {
    return typeof target === 'string' ? document.querySelector(target) : target;
  }

  function mount(target, options) {
    const host = resolve(target);
    if (!host) return null;
    const settings = options || {};
    const tabs = Array.from(host.querySelectorAll('[data-query-mode]'));
    const panels = Array.from(host.querySelectorAll('[data-query-panel]'));
    const runButton = host.querySelector('[data-query-run]');
    const summary = host.querySelector('[data-query-summary]');
    const sqlHost = host.querySelector('[data-query-sql-editor]');
    const useGenerated = host.querySelector('[data-use-generated-sql]');
    const editor = global.TopbaseCode.mountEditor(sqlHost, {
      language: 'sql',
      label: 'SQL 查询',
      placeholder: 'SELECT *\nFROM schema.table\nLIMIT 100;',
      onChange(value) {
        state.sqlDirty = true;
        if (typeof settings.onSQLChange === 'function') settings.onSQLChange(value);
      }
    });
    const state = { mode: 'visual', generatedSQL: '', sqlDirty: false, busy: false };

    function setMode(mode, focus) {
      if (mode !== 'visual' && mode !== 'sql') return;
      if (mode === 'sql' && !state.sqlDirty && state.generatedSQL) editor.set(state.generatedSQL);
      state.mode = mode;
      tabs.forEach(tab => {
        const active = tab.dataset.queryMode === mode;
        tab.classList.toggle('active', active);
        tab.setAttribute('aria-selected', active ? 'true' : 'false');
        tab.tabIndex = active ? 0 : -1;
      });
      panels.forEach(panel => { panel.hidden = panel.dataset.queryPanel !== mode; });
      if (runButton) runButton.textContent = mode === 'sql' ? '运行 SQL 并预览' : '运行并预览';
      host.dataset.mode = mode;
      if (focus && mode === 'sql') editor.focus();
      if (typeof settings.onModeChange === 'function') settings.onModeChange(mode);
    }

    tabs.forEach((tab, index) => {
      tab.onclick = () => setMode(tab.dataset.queryMode, true);
      tab.onkeydown = event => {
        if (!['ArrowLeft', 'ArrowRight'].includes(event.key)) return;
        event.preventDefault();
        const offset = event.key === 'ArrowRight' ? 1 : -1;
        const next = tabs[(index + offset + tabs.length) % tabs.length];
        setMode(next.dataset.queryMode, true);
        next.focus();
      };
    });
    if (useGenerated) useGenerated.onclick = () => {
      if (!state.generatedSQL) return;
      editor.set(state.generatedSQL);
      state.sqlDirty = true;
      editor.focus();
      if (typeof global.toast === 'function') global.toast('已载入可视化查询生成的 SQL');
    };
    if (runButton) runButton.onclick = async () => {
      if (state.busy || typeof settings.onRun !== 'function') return;
      state.busy = true;
      runButton.disabled = true;
      runButton.textContent = state.mode === 'sql' ? '正在运行 SQL…' : '正在运行…';
      try { await settings.onRun(state.mode); }
      finally {
        state.busy = false;
        runButton.disabled = false;
        runButton.textContent = state.mode === 'sql' ? '运行 SQL 并预览' : '运行并预览';
      }
    };
    setMode('visual');

    return {
      mode() { return state.mode; },
      setMode,
      sql() { return editor.value(); },
      setSQL(value, config) {
        editor.set(value || '');
        state.sqlDirty = !!(config && config.dirty);
      },
      setGeneratedSQL(value) { state.generatedSQL = value || ''; },
      setSummary(value) { if (summary) summary.textContent = value; }
    };
  }

  global.TopbaseQueryEditor = { mount };
})(window);
