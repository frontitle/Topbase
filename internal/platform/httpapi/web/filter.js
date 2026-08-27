(function (global) {
  function esc(s) {
    return String(s ?? '').replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }
  function pad(n) { return String(n).padStart(2, '0'); }
  function iso(d) { return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()); }
  function today() { var d = new Date(); d.setHours(0, 0, 0, 0); return d; }
  function addDays(d, n) { var x = new Date(d); x.setDate(x.getDate() + n); return x; }
  function startOfWeek(d) {
    var x = new Date(d);
    var day = x.getDay() || 7;
    x.setDate(x.getDate() - day + 1);
    return x;
  }

  var TEXT_OPS = [
    { op: 'eq', label: '是', values: 'many' },
    { op: 'neq', label: '不是', values: 'many' },
    { op: 'contains', label: '包含', values: 1 },
    { op: 'not_contains', label: '不包含', values: 1 },
    { op: 'starts_with', label: '开头是', values: 1 },
    { op: 'ends_with', label: '结尾是', values: 1 },
    { op: 'is_empty', label: '为空', values: 0 },
    { op: 'not_empty', label: '不为空', values: 0 }
  ];
  var NUMBER_OPS = [
    { op: 'eq', label: '等于', values: 'many' },
    { op: 'neq', label: '不等于', values: 'many' },
    { op: 'gt', label: '大于', values: 1 },
    { op: 'gte', label: '大于等于', values: 1 },
    { op: 'lt', label: '小于', values: 1 },
    { op: 'lte', label: '小于等于', values: 1 },
    { op: 'between', label: '介于两者之间', values: 2 },
    { op: 'is_null', label: '为空', values: 0 },
    { op: 'not_null', label: '不为空', values: 0 }
  ];
  var DATE_OPS = [
    { op: 'relative', label: '相对日期', values: 1 },
    { op: 'eq', label: '指定日期', values: 1 },
    { op: 'between', label: '日期范围', values: 2 },
    { op: 'gt', label: '之后', values: 1 },
    { op: 'lt', label: '之前', values: 1 },
    { op: 'is_null', label: '为空', values: 0 },
    { op: 'not_null', label: '不为空', values: 0 }
  ];
  var BOOL_OPS = [
    { op: 'eq', label: '是', values: 1 },
    { op: 'is_null', label: '为空', values: 0 },
    { op: 'not_null', label: '不为空', values: 0 }
  ];
  var RELATIVE = [
    { id: 'today', label: '今天' },
    { id: 'yesterday', label: '昨天' },
    { id: 'past_7', label: '过去 7 天' },
    { id: 'past_30', label: '过去 30 天' },
    { id: 'this_week', label: '本周' },
    { id: 'this_month', label: '本月' },
    { id: 'this_year', label: '今年' }
  ];

  function kindOf(col) {
    var semantic = (col.semantic_type || '').toLowerCase();
    var type = (col.data_type || '').toLowerCase();
    if (/date|time|timestamp|birthday/.test(semantic) || /date|time|timestamp/.test(type)) return 'date';
    if (semantic === 'quantity' || semantic === 'currency' || semantic === 'score' || semantic === 'percentage' || semantic === 'discount' || semantic === 'income' || semantic === 'latitude' || semantic === 'longitude') return 'number';
    if (/int|numeric|decimal|double|real|float|money|number/.test(type)) return 'number';
    if (semantic === 'boolean' || type === 'boolean' || type === 'bool') return 'boolean';
    return 'text';
  }
  function opsFor(kind) {
    if (kind === 'number') return NUMBER_OPS;
    if (kind === 'date') return DATE_OPS;
    if (kind === 'boolean') return BOOL_OPS;
    return TEXT_OPS;
  }
  function colName(col) { return col.display_name || col.name; }
  function findOp(kind, op) {
    return opsFor(kind).find(function (item) { return item.op === op; }) || opsFor(kind)[0];
  }
  function asList(value) {
    if (value == null || value === '') return [];
    return Array.isArray(value) ? value.map(String) : [String(value)];
  }
  function relativeRange(id) {
    var t = today();
    if (id === 'today') return [iso(t), iso(t)];
    if (id === 'yesterday') { var y = addDays(t, -1); return [iso(y), iso(y)]; }
    if (id === 'past_7') return [iso(addDays(t, -6)), iso(t)];
    if (id === 'past_30') return [iso(addDays(t, -29)), iso(t)];
    if (id === 'this_week') return [iso(startOfWeek(t)), iso(t)];
    if (id === 'this_month') return [iso(new Date(t.getFullYear(), t.getMonth(), 1)), iso(t)];
    if (id === 'this_year') return [iso(new Date(t.getFullYear(), 0, 1)), iso(t)];
    return [iso(t), iso(t)];
  }
  function coerce(kind, value) {
    if (kind !== 'number' || value == null) return value;
    if (Array.isArray(value)) return value.map(function (v) { var n = Number(v); return Number.isFinite(n) ? n : v; });
    var n = Number(value);
    return Number.isFinite(n) ? n : value;
  }
  function toQueryFilters(item, col) {
    var kind = kindOf(col);
    if (kind === 'date' && item.op === 'relative') {
      var range = relativeRange(item.value);
      return [
        { field: col.name, op: 'gte', value: range[0] },
        { field: col.name, op: 'lte', value: range[1] }
      ];
    }
    if (item.op === 'eq' && Array.isArray(item.value) && item.value.length > 1) {
      return [{ field: col.name, op: 'in', value: coerce(kind, item.value) }];
    }
    if (item.op === 'neq' && Array.isArray(item.value) && item.value.length > 1) {
      return [{ field: col.name, op: 'not_in', value: coerce(kind, item.value) }];
    }
    if ((item.op === 'is_null' || item.op === 'not_null' || item.op === 'is_empty' || item.op === 'not_empty')) {
      return [{ field: col.name, op: item.op }];
    }
    if (item.op === 'between') {
      return [{ field: col.name, op: 'between', value: coerce(kind, item.value) }];
    }
    var value = Array.isArray(item.value) && item.value.length === 1 ? item.value[0] : item.value;
    return [{ field: col.name, op: item.op, value: coerce(kind, value) }];
  }
  function fromQueryFilters(filters, columns) {
    filters = filters || [];
    var out = [];
    var skip = {};
    filters.forEach(function (f, i) {
      if (skip[i]) return;
      var col = columns.find(function (c) { return c.name === f.field; }) || { name: f.field };
      var next = filters[i + 1];
      if (f.op === 'gte' && next && next.field === f.field && next.op === 'lte') {
        skip[i + 1] = true;
        out.push({ field: f.field, op: 'between', value: [f.value, next.value], kind: kindOf(col) });
        return;
      }
      var op = f.op === 'in' ? 'eq' : (f.op === 'not_in' ? 'neq' : f.op);
      out.push({ field: f.field, op: op, value: f.value, kind: kindOf(col) });
    });
    return out;
  }
  function pillText(item, col) {
    var op = findOp(kindOf(col), item.op);
    var name = colName(col);
    if (!op || op.values === 0) return name + ' ' + (op ? op.label : item.op);
    if (item.op === 'relative') {
      var rel = RELATIVE.find(function (r) { return r.id === item.value; });
      return name + ' ' + (rel ? rel.label : item.value);
    }
    var vals = asList(item.value);
    if (item.op === 'between' && vals.length === 2) return name + ' 介于 ' + vals[0] + ' 和 ' + vals[1];
    return name + ' ' + (op ? op.label : item.op) + ' ' + vals.join('、');
  }

  global.TopbaseFilter = function (host, opts) {
    opts = opts || {};
    if (typeof host === 'string') host = document.querySelector(host);
    if (!host) return { setColumns: function () {}, setFilters: function () {}, destroy: function () {} };
    var columns = opts.columns || [];
    var items = fromQueryFilters(opts.filters || [], columns);
    var open = false;
    var step = 'columns';
    var draft = null;
    var values = [];
    var valueQuery = '';
    var colQuery = '';

    function emit() {
      var next = [];
      items.forEach(function (item) {
        var col = columns.find(function (c) { return c.name === item.field; });
        if (!col) return;
        next.push.apply(next, toQueryFilters(item, col));
      });
      if (opts.onChange) opts.onChange(next);
    }

    function closePop() {
      open = false;
      step = 'columns';
      draft = null;
      render();
    }
    function openAdd() {
      open = true;
      step = 'columns';
      draft = null;
      colQuery = '';
      render();
    }
    function openEdit(index) {
      open = true;
      step = 'value';
      draft = Object.assign({ index: index }, items[index]);
      valueQuery = '';
      loadValues();
      render();
    }
    function selectCol(name) {
      var col = columns.find(function (c) { return c.name === name; });
      if (!col) return;
      var kind = kindOf(col);
      draft = { field: name, op: opsFor(kind)[0].op, value: kind === 'date' ? 'past_7' : (kind === 'boolean' ? 'true' : []), kind: kind, index: draft && draft.index };
      if (kind === 'date') draft.op = 'relative';
      step = 'value';
      valueQuery = '';
      loadValues();
      render();
    }
    function loadValues() {
      values = [];
      if (!opts.fetchValues || !draft) return;
      var kind = kindOf(columns.find(function (c) { return c.name === draft.field; }) || {});
      if (kind !== 'text' && kind !== 'number') return;
      Promise.resolve(opts.fetchValues(draft.field)).then(function (rows) {
        values = (rows || []).map(String);
        if (open && step === 'value') render();
      }).catch(function () {});
    }
    function currentOp() {
      if (!draft) return null;
      var col = columns.find(function (c) { return c.name === draft.field; }) || { name: draft.field };
      return findOp(kindOf(col), draft.op);
    }
    function canApply() {
      var op = currentOp();
      if (!draft || !op) return false;
      if (op.values === 0) return true;
      if (op.values === 2) return asList(draft.value).length === 2 && asList(draft.value).every(Boolean);
      if (op.values === 'many') return asList(draft.value).length > 0;
      return draft.value != null && String(draft.value) !== '';
    }
    function applyDraft(keepOpen) {
      if (!canApply()) return;
      var next = { field: draft.field, op: draft.op, value: draft.value, kind: draft.kind };
      if (typeof draft.index === 'number') items[draft.index] = next;
      else items.push(next);
      emit();
      if (keepOpen) {
        draft = null;
        step = 'columns';
        render();
        return;
      }
      closePop();
    }
    function removeAt(index) {
      items.splice(index, 1);
      emit();
      render();
    }

    function renderColumns() {
      var q = colQuery.toLowerCase();
      var list = columns.filter(function (c) {
        return !q || (c.name + ' ' + colName(c)).toLowerCase().includes(q);
      });
      return '<input class="tb-filter-search" placeholder="搜索字段" value="' + esc(colQuery) + '">' +
        '<div class="tb-filter-cols">' +
        (list.length ? list.map(function (c) {
          return '<button type="button" class="tb-filter-col" data-col="' + esc(c.name) + '"><b>' + esc(colName(c)) + '</b><small>' + esc(kindOf(c) === 'date' ? '日期' : kindOf(c) === 'number' ? '数值' : kindOf(c) === 'boolean' ? '布尔' : '文本') + '</small></button>';
        }).join('') : '<div class="tb-filter-empty">没有匹配的字段。</div>') +
        '</div>';
    }
    function renderValue() {
      var col = columns.find(function (c) { return c.name === draft.field; }) || { name: draft.field };
      var kind = kindOf(col);
      var op = currentOp();
      var ops = opsFor(kind).map(function (item) {
        return '<option value="' + item.op + '"' + (item.op === draft.op ? ' selected' : '') + '>' + item.label + '</option>';
      }).join('');
      var body = '';
      if (op.values === 0) {
        body = '<p class="tb-filter-empty" style="padding:0">这条筛选不需要输入值。</p>';
      } else if (kind === 'boolean') {
        body = '<div class="tb-filter-bool">' +
          '<label><input type="radio" name="tb-bool" value="true"' + (String(draft.value) === 'true' ? ' checked' : '') + '> 真</label>' +
          '<label><input type="radio" name="tb-bool" value="false"' + (String(draft.value) === 'false' ? ' checked' : '') + '> 假</label>' +
          '</div>';
      } else if (kind === 'date' && draft.op === 'relative') {
        body = '<div class="tb-filter-dates">' + RELATIVE.map(function (r) {
          return '<button type="button" data-rel="' + r.id + '"' + (draft.value === r.id ? ' class="active"' : '') + '>' + r.label + '</button>';
        }).join('') + '</div>';
      } else if (kind === 'date' && draft.op === 'between') {
        var pair = asList(draft.value);
        body = '<div class="tb-filter-pair"><input type="date" data-bound="0" value="' + esc(pair[0] || '') + '"><input type="date" data-bound="1" value="' + esc(pair[1] || '') + '"></div>';
      } else if (kind === 'date') {
        body = '<input type="date" data-single value="' + esc(asList(draft.value)[0] || '') + '">';
      } else if (kind === 'number' && draft.op === 'between') {
        var nums = asList(draft.value);
        body = '<div class="tb-filter-pair"><input type="number" data-bound="0" value="' + esc(nums[0] || '') + '" placeholder="最小值"><input type="number" data-bound="1" value="' + esc(nums[1] || '') + '" placeholder="最大值"></div>';
      } else if (op.values === 'many') {
        var selected = asList(draft.value);
        var filtered = values.filter(function (v) { return !valueQuery || String(v).toLowerCase().includes(valueQuery.toLowerCase()); });
        body = '<input class="tb-filter-search" style="margin:0;width:100%" placeholder="搜索或勾选取值" data-value-search value="' + esc(valueQuery) + '">' +
          (filtered.length ? '<div class="tb-filter-values">' + filtered.map(function (v) {
            return '<label><input type="checkbox" data-pick="' + esc(v) + '"' + (selected.indexOf(String(v)) >= 0 ? ' checked' : '') + '> ' + esc(v || '（空）') + '</label>';
          }).join('') + '</div>' : '<div class="tb-filter-empty" style="padding:8px 0">没有现成取值，可在下方输入。</div>') +
          '<input type="text" data-custom placeholder="输入其他值后回车">';
      } else if (kind === 'number') {
        body = '<input type="number" data-single value="' + esc(asList(draft.value)[0] || '') + '" placeholder="输入数值">';
      } else {
        body = '<input type="text" data-single value="' + esc(asList(draft.value)[0] || '') + '" placeholder="输入文本">';
      }
      return '<header><button type="button" class="back" data-back>' + esc(colName(col)) + '</button></header>' +
        '<div class="tb-filter-body"><div class="tb-filter-ops"><select data-op>' + ops + '</select></div>' + body + '</div>' +
        '<div class="tb-filter-foot"><button type="button" class="ghost" data-add>再加一条</button><button type="button" class="apply" data-apply' + (canApply() ? '' : ' disabled') + '>' + (typeof draft.index === 'number' ? '更新筛选' : '应用筛选') + '</button></div>';
    }

    function render() {
      host.innerHTML = '<div class="tb-filter-bar">' +
        '<button type="button" class="tb-filter-btn' + (items.length ? ' active' : '') + (open ? ' open' : '') + '" data-open>筛选' +
          (items.length ? '<span class="count">' + items.length + '</span>' : '') +
        '</button>' +
        '<div class="tb-filter-pills">' + items.map(function (item, i) {
          var col = columns.find(function (c) { return c.name === item.field; }) || { name: item.field };
          return '<span class="tb-filter-pill" data-edit="' + i + '"><span>' + esc(pillText(item, col)) + '</span><button type="button" class="x" data-del="' + i + '" aria-label="移除">×</button></span>';
        }).join('') + '</div>' +
        (open ? '<div class="tb-filter-pop">' + (step === 'columns' || !draft ? renderColumns() : renderValue()) + '</div>' : '') +
      '</div>';

      var pop = host.querySelector('.tb-filter-pop');
      var btn = host.querySelector('[data-open]');
      if (pop && btn) {
        pop.style.top = (btn.offsetTop + btn.offsetHeight + 6) + 'px';
        pop.style.left = btn.offsetLeft + 'px';
      }

      host.querySelector('[data-open]').onclick = function (ev) {
        ev.stopPropagation();
        if (open) closePop(); else openAdd();
      };
      host.querySelectorAll('[data-edit]').forEach(function (el) {
        el.onclick = function (ev) {
          if (ev.target.closest('[data-del]')) return;
          ev.stopPropagation();
          openEdit(Number(el.dataset.edit));
        };
      });
      host.querySelectorAll('[data-del]').forEach(function (el) {
        el.onclick = function (ev) {
          ev.stopPropagation();
          removeAt(Number(el.dataset.del));
        };
      });
      var search = host.querySelector('.tb-filter-search:not([data-value-search])');
      if (search && step === 'columns') {
        search.oninput = function () { colQuery = search.value; render(); search.focus(); search.setSelectionRange(colQuery.length, colQuery.length); };
        search.focus();
      }
      host.querySelectorAll('[data-col]').forEach(function (el) {
        el.onclick = function (ev) { ev.stopPropagation(); selectCol(el.dataset.col); };
      });
      var back = host.querySelector('[data-back]');
      if (back) back.onclick = function (ev) { ev.stopPropagation(); step = 'columns'; draft = draft && typeof draft.index === 'number' ? draft : null; render(); };
      var opSel = host.querySelector('[data-op]');
      if (opSel) opSel.onchange = function () {
        draft.op = opSel.value;
        var op = currentOp();
        if (op.values === 0) draft.value = null;
        else if (op.values === 2) draft.value = asList(draft.value).slice(0, 2);
        else if (draft.op === 'relative') draft.value = draft.value && RELATIVE.some(function (r) { return r.id === draft.value; }) ? draft.value : 'past_7';
        else if (op.values === 'many') draft.value = asList(draft.value);
        else draft.value = asList(draft.value)[0] || '';
        render();
      };
      host.querySelectorAll('[data-rel]').forEach(function (el) {
        el.onclick = function (ev) { ev.stopPropagation(); draft.value = el.dataset.rel; render(); };
      });
      host.querySelectorAll('[name="tb-bool"]').forEach(function (el) {
        el.onchange = function () { draft.value = el.value; render(); };
      });
      host.querySelectorAll('[data-bound]').forEach(function (el) {
        el.oninput = function () {
          var pair = asList(draft.value);
          pair[Number(el.dataset.bound)] = el.value;
          draft.value = pair;
        };
        el.onchange = function () { render(); };
      });
      var single = host.querySelector('[data-single]');
      if (single) {
        single.oninput = function () { draft.value = single.value; };
        single.onchange = function () { render(); };
      }
      var valueSearch = host.querySelector('[data-value-search]');
      if (valueSearch) {
        valueSearch.oninput = function () { valueQuery = valueSearch.value; render(); };
      }
      host.querySelectorAll('[data-pick]').forEach(function (el) {
        el.onchange = function () {
          var selected = asList(draft.value);
          var v = el.dataset.pick;
          if (el.checked && selected.indexOf(v) < 0) selected.push(v);
          if (!el.checked) selected = selected.filter(function (x) { return x !== v; });
          draft.value = selected;
          host.querySelector('[data-apply]').disabled = !canApply();
        };
      });
      var custom = host.querySelector('[data-custom]');
      if (custom) custom.onkeydown = function (ev) {
        if (ev.key !== 'Enter') return;
        ev.preventDefault();
        var v = custom.value.trim();
        if (!v) return;
        var selected = asList(draft.value);
        if (selected.indexOf(v) < 0) selected.push(v);
        draft.value = selected;
        custom.value = '';
        render();
      };
      var add = host.querySelector('[data-add]');
      if (add) add.onclick = function (ev) { ev.stopPropagation(); applyDraft(true); };
      var apply = host.querySelector('[data-apply]');
      if (apply) apply.onclick = function (ev) { ev.stopPropagation(); applyDraft(false); };
      if (pop) pop.onclick = function (ev) { ev.stopPropagation(); };
    }

    function onDoc(ev) {
      if (!open) return;
      if (host.contains(ev.target)) return;
      closePop();
    }
    document.addEventListener('click', onDoc);
    render();
    return {
      setColumns: function (next) { columns = next || []; render(); },
      setFilters: function (next) { items = fromQueryFilters(next || [], columns); render(); },
      filters: function () {
        var out = [];
        items.forEach(function (item) {
          var col = columns.find(function (c) { return c.name === item.field; });
          if (col) out.push.apply(out, toQueryFilters(item, col));
        });
        return out;
      },
      destroy: function () {
        document.removeEventListener('click', onDoc);
        host.innerHTML = '';
      }
    };
  };
})(window);
