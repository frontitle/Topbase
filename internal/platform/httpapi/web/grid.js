(function (global) {
  function esc(s) {
    return String(s ?? '').replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }
  function isFiniteNumber(v) {
    if (v === null || v === undefined || v === '') return false;
    return Number.isFinite(Number(v));
  }
  function matchCell(val, filter) {
    if (global.TopbaseViz && TopbaseViz.matchFilter) return TopbaseViz.matchFilter(val, filter);
    if (!filter) return true;
    var raw = String(val ?? '');
    var f = filter.trim();
    if (!f) return true;
    var m = f.match(/^(>=|<=|!=|>|<|=)\s*(.*)$/);
    if (m) {
      var op = m[1], rhs = m[2];
      var n = Number(raw), r = Number(rhs);
      if (Number.isFinite(n) && Number.isFinite(r)) {
        if (op === '>') return n > r;
        if (op === '>=') return n >= r;
        if (op === '<') return n < r;
        if (op === '<=') return n <= r;
        if (op === '=') return n === r;
        if (op === '!=') return n !== r;
      }
      var a = raw.toLowerCase(), b = rhs.toLowerCase();
      if (op === '=' || op === '==') return a === b;
      if (op === '!=') return a !== b;
    }
    return raw.toLowerCase().includes(f.toLowerCase());
  }
  function uniqueValues(rows, index, limit) {
    var seen = Object.create(null), out = [];
    for (var i = 0; i < rows.length; i++) {
      var key = rows[i][index] == null ? '' : String(rows[i][index]);
      if (seen[key]) continue;
      seen[key] = true;
      out.push(key);
      if (out.length > limit) return null;
    }
    out.sort();
    return out;
  }
  var SENSITIVE_SEMANTICS = {
    Birthday: true, PersonName: true, MobilePhone: true, NationalID: true,
    BankCard: true, Email: true, Address: true, IPAddress: true
  };
  function maskMiddle(raw, head, tail) {
    var value = String(raw ?? '');
    if (value.length <= head + tail) return value.length <= 2 ? '*'.repeat(value.length) : value.slice(0, 1) + '*'.repeat(value.length - 2) + value.slice(-1);
    return value.slice(0, head) + '*'.repeat(Math.max(value.length - head - tail, 4)) + value.slice(-tail);
  }
  function maskSensitive(raw, semantic) {
    var value = String(raw ?? ''), type = String(semantic || '');
    if (!value) return value;
    if (type === 'MobilePhone') return maskMiddle(value, 3, 4);
    if (type === 'NationalID') return maskMiddle(value, 6, 4);
    if (type === 'BankCard') return maskMiddle(value, 4, 4);
    if (type === 'Email') {
      var at = value.indexOf('@');
      if (at < 0) return maskMiddle(value, 1, 1);
      var local = value.slice(0, at);
      return maskMiddle(local, 1, local.length > 2 ? 1 : 0) + value.slice(at);
    }
    if (type === 'PersonName') return value.slice(0, 1) + '*'.repeat(Math.max(value.length - 1, 1));
    if (type === 'Birthday') return value.replace(/(\d{4})[-/.年]?\d{1,2}[-/.月]?\d{1,2}.*/, '$1-**-**');
    if (type === 'IPAddress') return value.replace(/^([\da-f]+)[.:].*$/i, '$1.*.*.*');
    if (type === 'Address') return maskMiddle(value, Math.min(6, Math.max(2, Math.floor(value.length / 3))), 0);
    return maskMiddle(value, 2, 2);
  }

  global.TopbaseGrid = function (host, opts) {
    opts = opts || {};
    if (typeof host === 'string') host = document.querySelector(host);
    if (!host) return { setData: function () {}, destroy: function () {} };
    var popovers = [];
    function disposePopovers() { popovers.forEach(p => p.dispose()); popovers = []; }
    var state = {
      columns: opts.columns || [],
      rows: opts.rows || [],
      aliases: opts.aliases || {},
      types: opts.types || {},
      semanticTypes: opts.semanticTypes || {},
      descriptions: opts.descriptions || {},
      search: opts.search || '',
      filters: opts.filtersEnabled === false ? {} : Object.assign({}, opts.filters || {}),
      sort: opts.sort || '',
      dir: opts.dir || 'asc',
      group: opts.group || '',
      rowHeight: opts.rowHeight || 'normal',
      order: (opts.order || opts.columns || []).slice(),
      hidden: Object.assign({}, opts.hidden || {}),
      showFilters: false,
      revealSensitive: false
    };

    var columnIndexes = new Map();
    function indexColumns() {
      columnIndexes.clear();
      state.columns.forEach(function (name, i) {
        if (!columnIndexes.has(name)) columnIndexes.set(name, i);
      });
    }
    indexColumns();
    function colIndex(name) { return columnIndexes.get(name); }
    function visible() { return state.order.filter(function (c) { return !state.hidden[c]; }); }
    function sensitive(name) { return !!SENSITIVE_SEMANTICS[state.semanticTypes[name]]; }
    function displayValue(name, value) { return sensitive(name) && !state.revealSensitive ? maskSensitive(value, state.semanticTypes[name]) : value; }
    function moveColumn(name, direction) {
      var index = state.order.indexOf(name);
      var target = index + direction;
      if (index < 0 || target < 0 || target >= state.order.length) return;
      var item = state.order[index];
      state.order[index] = state.order[target];
      state.order[target] = item;
    }
    function typeMark(name) {
      var value = String(state.types[name] || '').toLowerCase();
      if (/int|numeric|decimal|double|real|float|money/.test(value)) return '123';
      if (/date|time|timestamp/.test(value)) return '日';
      if (/bool/.test(value)) return '✓';
      return 'T';
    }
    function notify() {
      if (!opts.onChange) return;
      opts.onChange({
        hidden: Object.assign({}, state.hidden),
        filters: Object.assign({}, state.filters),
        search: state.search,
        sort: state.sort,
        dir: state.dir,
        group: state.group,
        rowHeight: state.rowHeight,
        order: state.order.slice()
      });
    }

    function groupValue(rowIndex, fieldIndex) {
      return state.rows[rowIndex][fieldIndex] ?? '';
    }

    function displayedRowIndexes() {
      var q = (state.search || '').toLowerCase();
      var filters = Object.keys(state.filters).filter(function (name) {
        return state.filters[name] && columnIndexes.has(name);
      }).map(function (name) { return {index: colIndex(name), value: state.filters[name]}; });
      var rows = [];
      state.rows.forEach(function (row, index) {
        if (q) {
          var hit = false;
          for (var i = 0; i < row.length; i++) {
            if (state.hidden[state.columns[i]]) continue;
            if (String(row[i] ?? '').toLowerCase().includes(q)) { hit = true; break; }
          }
          if (!hit) return;
        }
        for (var c = 0; c < filters.length; c++) {
          if (!matchCell(row[filters[c].index], filters[c].value)) return;
        }
        rows.push(index);
      });
      if (state.sort) {
        var idx = colIndex(state.sort);
        var numeric = rows.every(function (index) {
          var value = state.rows[index][idx];
          return value == null || value === '' || isFiniteNumber(value);
        });
        rows.sort(function (a, b) {
          var va = state.rows[a][idx], vb = state.rows[b][idx];
          var cmp = 0;
          if (va == null && vb == null) cmp = 0;
          else if (va == null) cmp = -1;
          else if (vb == null) cmp = 1;
          else if (numeric) cmp = Number(va) - Number(vb);
          else cmp = String(va).localeCompare(String(vb), 'zh-CN', { numeric: true });
          return state.dir === 'desc' ? -cmp : cmp;
        });
      }
      if (state.group && columnIndexes.has(state.group)) {
        var groupIndex = colIndex(state.group);
        var groups = new Map();
        rows.forEach(function (index) {
          var value = groupValue(index, groupIndex);
          if (!groups.has(value)) groups.set(value, []);
          groups.get(value).push(index);
        });
        rows = Array.from(groups.values()).flat();
      }
      return rows;
    }

    function render(fieldAction) {
      var cols = visible();
      var visibleIndexes = cols.map(colIndex);
      var groupIndex = state.group ? colIndex(state.group) : undefined;
      var rows = displayedRowIndexes();
      var compact = !!opts.compact, dashboardOnly = !!opts.dashboardOnly, hideToolbar = !!opts.hideToolbar;
      var hasSensitive = cols.some(sensitive);
      var privacyButton = hasSensitive ? '<button class="tb-tool tb-sensitive-toggle' + (state.revealSensitive ? ' active' : '') + '" type="button" data-toggle-sensitive aria-pressed="' + (state.revealSensitive ? 'true' : 'false') + '">' + (state.revealSensitive ? '隐藏敏感数据' : '显示敏感数据') + '</button>' : '';
      var active = host.contains(document.activeElement) ? document.activeElement : null;
      var fieldPanel = host.querySelector('.tb-cols[open]');
      var fieldScroll = fieldPanel ? fieldPanel.querySelector('.tb-field-list').scrollTop : 0;
      var fieldSummaryFocused = fieldPanel && fieldPanel.querySelector('summary') === active;
      var viewport = host.querySelector('.tb-grid-scroll');
      var scrollLeft = viewport ? viewport.scrollLeft : 0, scrollTop = viewport ? viewport.scrollTop : 0;
      if (fieldPanel && !fieldAction && active && active.dataset) {
        if (active.dataset.toggleColumn !== undefined) fieldAction = {column:active.dataset.toggleColumn};
        else if (active.dataset.moveField) fieldAction = {column:active.dataset.col, move:active.dataset.moveField};
      }
      var restore = null;
      if (active && active.classList && active.classList.contains('tb-search')) {
        restore = { search: true, start: active.selectionStart, end: active.selectionEnd };
      } else if (active && active.dataset && active.dataset.filter) {
        restore = { filter: active.dataset.filter, start: active.selectionStart, end: active.selectionEnd };
      }
      disposePopovers();
      host.innerHTML = '<div class="tb-grid tb-row-' + state.rowHeight + (compact ? ' compact' : '') + (dashboardOnly ? ' dashboard-only' : '') + '">' +
		(hideToolbar ? (privacyButton ? '<div class="tb-grid-bar tb-privacy-bar"><span class="tb-privacy-hint">敏感字段已默认脱敏，搜索仍使用原始数据</span>' + privacyButton + '</div>' : '') : '<div class="tb-grid-bar">' +
	        '<span class="tb-view-name"><span class="tb-view-dot">▦</span> 表格视图</span>' +
	        '<span class="tb-toolbar-divider"></span>' +
          (compact ? '' : '<details class="tb-group"><summary>分组' + (state.group ? ' · ' + esc(state.aliases[state.group] || state.group) : '') + '</summary><div><label>按字段分组<select class="tb-select" data-group><option value="">不分组</option>' + state.columns.map(function (c) { return '<option value="' + esc(c) + '"' + (state.group === c ? ' selected' : '') + '>' + esc(state.aliases[c] || c) + '</option>'; }).join('') + '</select></label></div></details>') +
	        (compact ? '' : '<details class="tb-row-height"><summary>行高</summary><div><label><select class="tb-select" data-row-height aria-label="行高"><option value="compact"' + (state.rowHeight === 'compact' ? ' selected' : '') + '>紧凑</option><option value="normal"' + (state.rowHeight === 'normal' ? ' selected' : '') + '>标准</option><option value="loose"' + (state.rowHeight === 'loose' ? ' selected' : '') + '>宽松</option></select></label></div></details>') +
	        (opts.filtersEnabled === false ? '' : '<button class="tb-tool" type="button" data-toggle-filters>筛选</button>') +
          '<button class="tb-tool" type="button" data-clear-sort' + (state.sort ? '' : ' disabled') + '>排序</button>' +
          '<input class="tb-search tb-input" aria-label="搜索记录" placeholder="搜索记录">' +
          '<span class="tb-count">显示 ' + rows.length + ' / ' + state.rows.length + ' 行</span>' +
          privacyButton +
          (compact ? '' : '<details class="tb-cols"' + (fieldPanel ? ' open' : '') + '><summary>字段</summary><div class="tb-field-list">' +
            state.order.map(function (c, index) {
              var visibleLabel = state.hidden[c] ? '显示' : '隐藏';
              return '<div class="tb-field-item" draggable="true" data-field="' + esc(c) + '" title="' + esc(state.descriptions[c] || '') + '"><span class="tb-field-grip" aria-hidden="true">⋮⋮</span><span class="tb-type">' + typeMark(c) + '</span><span class="tb-field-name">' + esc(state.aliases[c] || c) + '</span><span class="tb-field-actions"><button type="button" data-move-field="up" data-col="' + esc(c) + '" aria-label="上移字段"' + (index === 0 ? ' disabled' : '') + '>↑</button><button type="button" data-move-field="down" data-col="' + esc(c) + '" aria-label="下移字段"' + (index === state.order.length - 1 ? ' disabled' : '') + '>↓</button><button type="button" data-toggle-column="' + esc(c) + '" aria-label="' + visibleLabel + '字段">' + visibleLabel + '</button></span></div>';
            }).join('') + '</div></details>') +
        '</div>') +
        '<div class="tb-grid-scroll"><table><thead><tr>' +
          (dashboardOnly ? '' : '<th class="tb-row-number">#</th>') +
          cols.map(function (c) {
            var mark = state.sort === c ? (state.dir === 'desc' ? ' ↓' : ' ↑') : '';
            return '<th' + (dashboardOnly ? '' : ' data-sort="' + esc(c) + '"') + ' title="' + esc(state.descriptions[c] || '') + '"><span class="tb-type">' + typeMark(c) + '</span>' + esc(state.aliases[c] || c) + (dashboardOnly ? '' : mark) + (state.aliases[c] ? '<small>' + esc(c) + '</small>' : '') + '</th>';
          }).join('') +
	    '</tr>' + (opts.filtersEnabled !== false && state.showFilters ? '<tr class="tb-filters"><th class="tb-row-number"></th>' +
          cols.map(function (c) {
            var idx = colIndex(c);
            var uniques = uniqueValues(state.rows, idx, 16);
            var val = esc(state.filters[c] || '');
            var label = esc('筛选' + (state.aliases[c] || c));
            if (uniques && uniques.length) {
              return '<th><select class="tb-select" aria-label="' + label + '" data-filter="' + esc(c) + '"><option value="">全部</option>' +
                uniques.map(function (u) {
                  var stored = '=' + u;
                  return '<option value="' + esc(stored) + '"' + (state.filters[c] === stored ? ' selected' : '') + '>' + (u === '' ? '（空）' : esc(displayValue(c, u))) + '</option>';
                }).join('') + '</select></th>';
            }
            return '<th><input class="tb-input" aria-label="' + label + '" data-filter="' + esc(c) + '" value="' + val + '" placeholder="包含 / > = <"></th>';
          }).join('') + '</tr>' : '') + '</thead><tbody>' +
          (rows.length ? rows.map(function (sourceIndex, rowIndex) {
            var row = state.rows[sourceIndex];
            var groupCell = groupIndex === undefined ? null : groupValue(sourceIndex, groupIndex);
            var previous = rowIndex && groupIndex !== undefined ? groupValue(rows[rowIndex - 1], groupIndex) : undefined;
            var columnCount = Math.max(cols.length + (dashboardOnly ? 0 : 1), 1);
            var groupHeader = groupIndex !== undefined && groupCell !== previous ? '<tr class="tb-group-row"><td colspan="' + columnCount + '"><span class="tb-caret" aria-hidden="true"></span>' + esc(state.aliases[state.group] || state.group) + '：<b>' + esc(groupCell === '' ? '（空）' : displayValue(state.group, groupCell)) + '</b></td></tr>' : '';
            return groupHeader + '<tr>' + (dashboardOnly ? '' : '<td class="tb-row-number">' + (sourceIndex + 1) + '</td>') + cols.map(function (c, columnIndex) {
              var v = row[visibleIndexes[columnIndex]];
              if (v === null || v === undefined || v === '') return '<td class="null">—</td>';
              var cls = isFiniteNumber(v) ? ' num' : '';
              var shown = displayValue(c, v);
              return '<td class="' + cls + (sensitive(c) ? ' sensitive' : '') + '" title="' + esc(shown) + '">' + esc(shown) + '</td>';
            }).join('') + '</tr>';
          }).join('') : '<tr><td class="empty" colspan="' + Math.max(cols.length + (dashboardOnly ? 0 : 1), 1) + '">没有匹配的行。试试清空筛选，或用 >100、=已完成 这样的条件。</td></tr>') +
        '</tbody></table></div></div>';

      host.querySelectorAll('.tb-grid-bar details').forEach(function (details) {
        var summary = details.querySelector('summary');
        var popup = global.TopbaseUI.popover(summary, summary.nextElementSibling, {
          type: 'dialog',
          onOpen: () => { details.open = true; },
          onClose: () => { details.open = false; }
        });
        popovers.push(popup);
        if (details.open) popup.open({ moveFocus: false });
      });
      var search = host.querySelector('.tb-search');
      if (search) {
        search.value = state.search;
        search.oninput = function () { state.search = search.value; render(); notify(); };
      }
	  var filterButton = host.querySelector('[data-toggle-filters]');
	  if (filterButton) filterButton.onclick = function () { state.showFilters = !state.showFilters; render(); };
      var sensitiveButton = host.querySelector('[data-toggle-sensitive]');
      if (sensitiveButton) sensitiveButton.onclick = function () { state.revealSensitive = !state.revealSensitive; render(); };
      var clearSort = host.querySelector('[data-clear-sort]');
      if (clearSort) clearSort.onclick = function () { state.sort = ''; state.dir = 'asc'; render(); notify(); };
      var groupSelect = host.querySelector('[data-group]');
      if (groupSelect) groupSelect.onchange = function () { state.group = groupSelect.value; render(); host.querySelector('.tb-group summary').focus(); notify(); };
      var heightSelect = host.querySelector('[data-row-height]');
      if (heightSelect) heightSelect.onchange = function () { state.rowHeight = heightSelect.value; render(); host.querySelector('.tb-row-height summary').focus(); notify(); };
      host.querySelectorAll('[data-sort]').forEach(function (th) {
        th.onclick = function () {
          var col = th.dataset.sort;
          if (state.sort === col) state.dir = state.dir === 'asc' ? 'desc' : 'asc';
          else { state.sort = col; state.dir = 'asc'; }
          render();
          notify();
        };
      });
      host.querySelectorAll('[data-filter]').forEach(function (input) {
        input.onchange = input.oninput = function () {
          state.filters[input.dataset.filter] = input.value;
          render();
          notify();
        };
      });
      host.querySelectorAll('[data-toggle-column]').forEach(function (button) {
        button.onclick = function () {
          var column = button.dataset.toggleColumn;
          if (state.hidden[column]) delete state.hidden[column];
          else state.hidden[column] = true;
          render({column: column});
          notify();
        };
      });
      host.querySelectorAll('[data-move-field]').forEach(function (button) {
        button.onclick = function () {
          moveColumn(button.dataset.col, button.dataset.moveField === 'up' ? -1 : 1);
          render({column: button.dataset.col, move: button.dataset.moveField});
          notify();
        };
      });
      host.querySelectorAll('.tb-field-item[draggable]').forEach(function (item) {
        item.ondragstart = function (event) { event.dataTransfer.setData('text/plain', item.dataset.field); };
        item.ondragover = function (event) { event.preventDefault(); item.classList.add('drag-over'); };
        item.ondragleave = function () { item.classList.remove('drag-over'); };
        item.ondrop = function (event) {
          event.preventDefault();
          item.classList.remove('drag-over');
          var from = event.dataTransfer.getData('text/plain'), to = item.dataset.field;
          var oldIndex = state.order.indexOf(from), newIndex = state.order.indexOf(to);
          if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return;
          state.order.splice(oldIndex, 1);
          state.order.splice(newIndex, 0, from);
          render({column: from});
          notify();
        };
      });
      viewport = host.querySelector('.tb-grid-scroll');
      if (viewport) { viewport.scrollLeft = scrollLeft; viewport.scrollTop = scrollTop; }
      if (fieldPanel) {
        var panel = host.querySelector('.tb-cols');
        panel.querySelector('.tb-field-list').scrollTop = fieldScroll;
        if (fieldAction) {
          var item = Array.from(panel.querySelectorAll('[data-field]')).find(function (el) { return el.dataset.field === fieldAction.column; });
          var button = fieldAction.move && item && item.querySelector('[data-move-field="' + fieldAction.move + '"]:not(:disabled)');
          var target = button || (item && item.querySelector('[data-toggle-column]')) || panel.querySelector('summary');
          target.focus({preventScroll: true});
          if (item) {
            var list = panel.querySelector('.tb-field-list');
            var itemRect = item.getBoundingClientRect(), listRect = list.getBoundingClientRect();
            if (itemRect.top < listRect.top) list.scrollTop += itemRect.top - listRect.top;
            else if (itemRect.bottom > listRect.bottom) list.scrollTop += itemRect.bottom - listRect.bottom;
          }
        } else if (fieldSummaryFocused) {
          panel.querySelector('summary').focus({preventScroll:true});
        }
      }
      if (restore && restore.search) {
        search.focus();
        if (restore.start != null) try { search.setSelectionRange(restore.start, restore.end); } catch (e) {}
      } else if (restore && restore.filter) {
        var el = Array.from(host.querySelectorAll('[data-filter]')).find(function (input) { return input.dataset.filter === restore.filter; });
        if (el) {
          el.focus();
          if (el.setSelectionRange && restore.start != null) try { el.setSelectionRange(restore.start, restore.end); } catch (e) {}
        }
      }
    }

    render();
    return {
      setData: function (next) {
        state.columns = next.columns || [];
        indexColumns();
        var previousOrder = new Set(state.order);
        state.order = state.order.filter(function (column) { return columnIndexes.has(column); }).concat(state.columns.filter(function (column) { return !previousOrder.has(column); }));
        if (!columnIndexes.has(state.group)) state.group = '';
        if (!columnIndexes.has(state.sort)) state.sort = '';
        state.rows = next.rows || [];
        state.aliases = next.aliases || {};
        state.types = next.types || state.types;
        state.semanticTypes = next.semanticTypes || state.semanticTypes;
        state.descriptions = next.descriptions || state.descriptions;
        render();
      },
      destroy: function () { disposePopovers(); host.innerHTML = ''; }
    };
  };
})(window);
