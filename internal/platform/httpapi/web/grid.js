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
    var seen = {}, out = [];
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

  global.TopbaseGrid = function (host, opts) {
    opts = opts || {};
    if (typeof host === 'string') host = document.querySelector(host);
    if (!host) return { setData: function () {}, destroy: function () {} };
    var state = {
      columns: opts.columns || [],
      rows: opts.rows || [],
      aliases: opts.aliases || {},
      types: opts.types || {},
      descriptions: opts.descriptions || {},
      search: opts.search || '',
      filters: opts.filtersEnabled === false ? {} : Object.assign({}, opts.filters || {}),
      sort: opts.sort || '',
      dir: opts.dir || 'asc',
      hidden: Object.assign({}, opts.hidden || {}),
      showFilters: false
    };

    function colIndex(name) { return state.columns.indexOf(name); }
    function visible() { return state.columns.filter(function (c) { return !state.hidden[c]; }); }
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
        dir: state.dir
      });
    }

    function filteredRows() {
      var q = (state.search || '').toLowerCase();
      var rows = state.rows.filter(function (row) {
        if (q) {
          var hit = false;
          for (var i = 0; i < row.length; i++) {
            if (state.hidden[state.columns[i]]) continue;
            if (String(row[i] ?? '').toLowerCase().includes(q)) { hit = true; break; }
          }
          if (!hit) return false;
        }
        for (var c = 0; c < state.columns.length; c++) {
          var name = state.columns[c];
          if (!matchCell(row[c], state.filters[name])) return false;
        }
        return true;
      });
      if (state.sort) {
        var idx = colIndex(state.sort);
        var numeric = rows.length && rows.every(function (r) { return r[idx] == null || r[idx] === '' || isFiniteNumber(r[idx]); });
        rows = rows.slice().sort(function (a, b) {
          var va = a[idx], vb = b[idx];
          var cmp = 0;
          if (va == null && vb == null) cmp = 0;
          else if (va == null) cmp = -1;
          else if (vb == null) cmp = 1;
          else if (numeric) cmp = Number(va) - Number(vb);
          else cmp = String(va).localeCompare(String(vb), 'zh-CN', { numeric: true });
          return state.dir === 'desc' ? -cmp : cmp;
        });
      }
      return rows;
    }

    function render() {
      var cols = visible();
      var rows = filteredRows();
      var compact = !!opts.compact;
      var active = host.contains(document.activeElement) ? document.activeElement : null;
      var restore = null;
      if (active && active.classList && active.classList.contains('tb-search')) {
        restore = { search: true, start: active.selectionStart, end: active.selectionEnd };
      } else if (active && active.dataset && active.dataset.filter) {
        restore = { filter: active.dataset.filter, start: active.selectionStart, end: active.selectionEnd };
      }
      host.innerHTML = '<div class="tb-grid' + (compact ? ' compact' : '') + '">' +
	      '<div class="tb-grid-bar">' +
	        '<span class="tb-view-name">表格视图</span>' +
	        (opts.filtersEnabled === false ? '' : '<button class="tb-tool" type="button" data-toggle-filters>筛选</button>') +
          '<span class="tb-sort-tip">点击字段名排序</span>' +
          '<input class="tb-search" placeholder="搜索记录">' +
          '<span class="tb-count">显示 ' + rows.length + ' / ' + state.rows.length + ' 行</span>' +
          (compact ? '' : '<details class="tb-cols"><summary>字段</summary>' +
            state.columns.map(function (c) {
              return '<label title="' + esc(state.descriptions[c] || '') + '"><input type="checkbox" data-col="' + esc(c) + '"' + (state.hidden[c] ? '' : ' checked') + '><span class="tb-type">' + typeMark(c) + '</span>' + esc(state.aliases[c] || c) + '</label>';
            }).join('') + '</details>') +
        '</div>' +
        '<div class="tb-grid-scroll"><table><thead><tr>' +
          '<th class="tb-row-number">#</th>' +
          cols.map(function (c) {
            var mark = state.sort === c ? (state.dir === 'desc' ? ' ↓' : ' ↑') : '';
            return '<th data-sort="' + esc(c) + '" title="' + esc(state.descriptions[c] || '') + '"><span class="tb-type">' + typeMark(c) + '</span>' + esc(state.aliases[c] || c) + mark + (state.aliases[c] ? '<small>' + esc(c) + '</small>' : '') + '</th>';
          }).join('') +
	    '</tr>' + (opts.filtersEnabled !== false && state.showFilters ? '<tr class="tb-filters"><th class="tb-row-number"></th>' +
          cols.map(function (c) {
            var idx = colIndex(c);
            var uniques = uniqueValues(state.rows, idx, 16);
            var val = esc(state.filters[c] || '');
            if (uniques && uniques.length) {
              return '<th><select data-filter="' + esc(c) + '"><option value="">全部</option>' +
                uniques.map(function (u) {
                  var stored = '=' + u;
                  return '<option value="' + esc(stored) + '"' + (state.filters[c] === stored ? ' selected' : '') + '>' + (u === '' ? '（空）' : esc(u)) + '</option>';
                }).join('') + '</select></th>';
            }
            return '<th><input data-filter="' + esc(c) + '" value="' + val + '" placeholder="包含 / > = <"></th>';
          }).join('') + '</tr>' : '') + '</thead><tbody>' +
          (rows.length ? rows.map(function (row) {
            return '<tr><td class="tb-row-number">' + (state.rows.indexOf(row)+1) + '</td>' + cols.map(function (c) {
              var v = row[colIndex(c)];
              if (v === null || v === undefined || v === '') return '<td class="null">—</td>';
              var cls = isFiniteNumber(v) ? ' num' : '';
              return '<td class="' + cls + '" title="' + esc(v) + '">' + esc(v) + '</td>';
            }).join('') + '</tr>';
          }).join('') : '<tr><td class="empty" colspan="' + Math.max(cols.length+1, 1) + '">没有匹配的行。试试清空筛选，或用 >100、=已完成 这样的条件。</td></tr>') +
        '</tbody></table></div></div>';

      var search = host.querySelector('.tb-search');
      search.value = state.search;
      search.oninput = function () { state.search = search.value; render(); notify(); };
	  var filterButton = host.querySelector('[data-toggle-filters]');
	  if (filterButton) filterButton.onclick = function () { state.showFilters = !state.showFilters; render(); };
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
      host.querySelectorAll('.tb-cols input[data-col]').forEach(function (box) {
        box.onchange = function () {
          if (box.checked) delete state.hidden[box.dataset.col];
          else state.hidden[box.dataset.col] = true;
          render();
          notify();
        };
      });
      if (restore && restore.search) {
        search.focus();
        if (restore.start != null) try { search.setSelectionRange(restore.start, restore.end); } catch (e) {}
      } else if (restore && restore.filter) {
        var el = host.querySelector('[data-filter="' + restore.filter + '"]');
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
        state.rows = next.rows || [];
        state.aliases = next.aliases || {};
        state.types = next.types || state.types;
        state.descriptions = next.descriptions || state.descriptions;
        render();
      },
      destroy: function () { host.innerHTML = ''; }
    };
  };
})(window);
