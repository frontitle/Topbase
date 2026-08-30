(function (global) {
  var TYPES = [
    { id: 'table', label: '表格' },
    { id: 'scalar', label: '数字' },
    { id: 'line', label: '折线图' },
    { id: 'area', label: '面积图' },
    { id: 'bar', label: '柱状图' },
    { id: 'row', label: '条形图' },
    { id: 'scatter', label: '散点图' },
    { id: 'pie', label: '饼图' }
  ];
  var COLORS = ['#509EE3', '#88BF4D', '#A989C5', '#EF8C8C', '#F9D45C', '#F2A86F', '#98D9D9', '#7172AD', '#ED8535', '#00A989', '#C5A3E6', '#FF8A80'];

  function esc(s) {
    return String(s ?? '').replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }
  function isNum(v) {
    if (v === null || v === undefined || v === '') return false;
    return Number.isFinite(Number(v));
  }
  function toNum(v) {
    var n = Number(v);
    return Number.isFinite(n) ? n : NaN;
  }
  function colIndex(columns, name) { return (columns || []).indexOf(name); }
  function looksDate(v) { return /^\d{4}-\d{2}-\d{2}/.test(String(v || '')); }
  function axisSpec(spec, which) { return (spec && spec[which]) || {}; }
  function seriesStyle(spec, name) { return ((spec && spec.series) || {})[name] || {}; }
  function columnStyle(spec, name) { return ((spec && spec.columns) || {})[name] || {}; }

  function formatNumber(v, spec, opts) {
    opts = opts || {};
    if (v === null || v === undefined || v === '') return '—';
    var n = Number(v);
    if (!Number.isFinite(n)) return String(v);
    spec = spec || {};
    if (isNum(spec.multiply)) n *= Number(spec.multiply);
    var style = spec.number_style || '';
    var vf = spec.value_format || 'auto';
    var dec = spec.decimals == null || spec.decimals === '' ? null : Number(spec.decimals);
    if (opts.percent || style === 'percent') {
      var pct = opts.alreadyPercent ? n : n * 100;
      var pt = dec != null ? pct.toFixed(dec) : (Math.abs(pct) >= 100 ? String(Math.round(pct)) : pct.toFixed(1));
      return pt + '%';
    }
    if (style === 'scientific') return n.toExponential(dec != null ? dec : 2);
    var compact = vf === 'compact' || (vf === 'auto' && !opts.full && Math.abs(n) >= 10000);
    var text;
    if (compact && Math.abs(n) >= 1e8) {
      var yi = n / 1e8;
      text = (dec != null ? yi.toFixed(dec) : yi.toFixed(yi >= 10 ? 1 : 2)) + '亿';
    } else if (compact && Math.abs(n) >= 10000) {
      var wan = n / 10000;
      text = (dec != null ? wan.toFixed(dec) : wan.toFixed(wan >= 10 ? 1 : 2)) + '万';
    } else if (dec != null) {
      text = n.toLocaleString('zh-CN', { minimumFractionDigits: dec, maximumFractionDigits: dec });
    } else if (Math.abs(n) >= 1000) {
      text = n.toLocaleString('zh-CN', { maximumFractionDigits: 2 });
    } else if (Math.abs(n) < 1 && n !== 0) {
      text = String(Number(n.toPrecision(4)));
    } else {
      text = String(Math.round(n * 100) / 100);
    }
    if (style === 'currency' && !spec.prefix) text = '¥' + text;
    return (spec.prefix || '') + text + (spec.suffix || '');
  }
  function fmt(v) { return formatNumber(v, null); }

  function matchFilter(val, filter) {
    if (!filter) return true;
    var raw = String(val ?? '');
    var f = String(filter).trim();
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
  function tableViewState(spec) {
    var aliases = {}, hidden = {};
    Object.keys((spec && spec.columns) || {}).forEach(function (k) {
      var c = spec.columns[k] || {};
      if (c.title) aliases[k] = c.title;
      if (c.visible === false) hidden[k] = true;
    });
    return {
      aliases: aliases,
      hidden: hidden,
      filters: {},
      search: (spec && spec.search) || '',
      sort: (spec && spec.sort) || '',
      dir: (spec && spec.sort_dir) || 'asc'
    };
  }
  function project(columns, rows, spec) {
    columns = columns || [];
    rows = rows || [];
    spec = spec || {};
    var view = tableViewState(spec);
    var visible = columns.filter(function (c) { return !view.hidden[c]; });
    if (!visible.length) visible = columns.slice();
    var q = (view.search || '').toLowerCase();
    var filtered = rows.filter(function (row) {
      if (q) {
        var hit = false;
        for (var i = 0; i < columns.length; i++) {
          if (view.hidden[columns[i]]) continue;
          if (String(row[i] ?? '').toLowerCase().includes(q)) { hit = true; break; }
        }
        if (!hit) return false;
      }
      for (var i = 0; i < columns.length; i++) {
        if (!matchFilter(row[i], view.filters[columns[i]])) return false;
      }
      return true;
    });
    if (view.sort && columns.indexOf(view.sort) >= 0) {
      var idx = columns.indexOf(view.sort);
      var numeric = filtered.length && filtered.every(function (r) { return r[idx] == null || r[idx] === '' || isNum(r[idx]); });
      filtered = filtered.slice().sort(function (a, b) {
        var va = a[idx], vb = b[idx], cmp = 0;
        if (va == null && vb == null) cmp = 0;
        else if (va == null) cmp = -1;
        else if (vb == null) cmp = 1;
        else if (numeric) cmp = Number(va) - Number(vb);
        else cmp = String(va).localeCompare(String(vb), 'zh-CN', { numeric: true });
        return view.dir === 'desc' ? -cmp : cmp;
      });
    }
    var indexes = visible.map(function (c) { return columns.indexOf(c); });
    return {
      columns: visible,
      rows: filtered.map(function (row) { return indexes.map(function (i) { return row[i]; }); }),
      aliases: view.aliases
    };
  }
  function classify(columns, rows) {
    var dims = [], metrics = [];
    (columns || []).forEach(function (name, i) {
      var numeric = (rows || []).length && rows.every(function (r) { return r[i] == null || r[i] === '' || isNum(r[i]); });
      var dateish = (rows || []).some(function (r) { return looksDate(r[i]); });
      if (numeric && !dateish) metrics.push(name);
      else dims.push(name);
    });
    return { dims: dims, metrics: metrics };
  }
  function infer(columns, rows, queryir, saved) {
    var kind = classify(columns, rows);
    var inferred = { type: 'table', x: kind.dims[0] || columns[0] || '', y: kind.metrics.slice() };
    if (queryir) {
      var ys = (queryir.aggregations || []).map(function (a) { return a.alias || String(a.fn || '').toLowerCase(); }).filter(Boolean);
      var gb = (queryir.group_by || [])[0];
      if (!gb && ys.length === 1) inferred = { type: 'scalar', y: ys };
      else if (gb && gb.temporal) inferred = { type: 'line', x: (gb.field || '') + '_' + String(gb.temporal).toLowerCase(), y: ys };
      else if (gb && ys.length) inferred = { type: 'bar', x: gb.field, y: ys };
    } else if (kind.dims.length && kind.metrics.length) {
      var dateDim = kind.dims.find(function (name) {
        var i = colIndex(columns, name);
        return (rows || []).some(function (r) { return looksDate(r[i]); });
      });
      inferred = { type: dateDim ? 'line' : 'bar', x: dateDim || kind.dims[0], y: kind.metrics.slice(0, 4) };
    } else if (kind.metrics.length === 1 && (rows || []).length <= 1) {
      inferred = { type: 'scalar', y: kind.metrics.slice() };
    }
    return merge(saved, inferred, columns, rows);
  }
  function merge(saved, inferred, columns, rows) {
    inferred = inferred || { type: 'table' };
    var out = Object.assign({ type: inferred.type, x: inferred.x, y: (inferred.y || []).slice() }, saved || {});
    if (!out.type) out.type = inferred.type || 'table';
    var kind = classify(columns, rows);
    if (!out.x) out.x = inferred.x || kind.dims[0] || (columns || [])[0] || '';
    if (!out.y || !out.y.length) out.y = (inferred.y && inferred.y.length ? inferred.y : kind.metrics).slice();
    out.y = (out.y || []).filter(function (name) { return (columns || []).indexOf(name) >= 0; });
    if (out.x && (columns || []).indexOf(out.x) < 0) out.x = inferred.x || kind.dims[0] || '';
    if (out.breakout && (columns || []).indexOf(out.breakout) < 0) out.breakout = '';
    return out;
  }
  function stackingOf(spec) {
    if (spec.stacking === 'stack' || spec.stacking === 'percent') return spec.stacking;
    if (spec.stacked) return 'stack';
    return '';
  }
  function colorOf(spec, name, index) {
    var st = seriesStyle(spec, name);
    if (st.color) return st.color;
    if (name === '其他' && spec.other_color) return spec.other_color;
    return COLORS[index % COLORS.length];
  }
  function titleOf(spec, name) { return seriesStyle(spec, name).title || name; }
  function visibleOf(spec, name) { return seriesStyle(spec, name).visible !== false; }
  function displayOf(spec, st) {
    if (st && st.display) return st.display;
    if (spec.type === 'area') return 'area';
    if (spec.type === 'bar' || spec.type === 'row') return 'bar';
    if (spec.type === 'scatter') return 'scatter';
    return 'line';
  }
  function interpOf(spec, st) {
    if (st && st.interpolate) return st.interpolate;
    if (spec.interpolation) return spec.interpolation;
    return spec.smooth === true ? 'cardinal' : 'linear';
  }
  function missingOf(spec, st) { return (st && st.missing) || spec.missing || 'interpolate'; }
  function lineWidth(st) {
    var size = String((st && st.line_size) || 'M').toUpperCase();
    if (size === 'S') return 1.5;
    if (size === 'L') return 4;
    return 2.5;
  }
  function showDots(spec, st, n) {
    var m = (st && st.markers) || spec.markers || 'auto';
    if (m === 'on' || m === true) return true;
    if (m === 'off' || m === false) return false;
    return n < 24;
  }
  function goalOn(spec) {
    if (!isNum(spec.goal)) return false;
    if (spec.show_goal === false) return false;
    return true;
  }
  function legendOn(spec, n) {
    if (spec.legend === 'off' || spec.show_legend === false) return false;
    return n > 0;
  }
  function percentMode(spec) {
    if (spec.percent) return spec.percent;
    return spec.show_labels ? 'inside' : 'legend';
  }

  function seriesValues(columns, rows, spec) {
    var xi = colIndex(columns, spec.x);
    var bi = spec.breakout ? colIndex(columns, spec.breakout) : -1;
    if (bi >= 0 && spec.y && spec.y[0]) {
      var yi = colIndex(columns, spec.y[0]);
      var labels = [], labelIndex = {}, seriesNames = [], seriesIndex = {};
      (rows || []).forEach(function (r) {
        var lab = r[xi] == null ? '' : String(r[xi]);
        if (labelIndex[lab] == null) { labelIndex[lab] = labels.length; labels.push(lab); }
        var name = r[bi] == null ? '空' : String(r[bi]);
        if (seriesIndex[name] == null) { seriesIndex[name] = seriesNames.length; seriesNames.push(name); }
      });
      var series = seriesNames.map(function (name) {
        return { name: name, data: labels.map(function () { return NaN; }) };
      });
      (rows || []).forEach(function (r) {
        var lab = r[xi] == null ? '' : String(r[xi]);
        var name = r[bi] == null ? '空' : String(r[bi]);
        var v = toNum(r[yi]);
        var li = labelIndex[lab], si = seriesIndex[name];
        if (Number.isFinite(series[si].data[li])) series[si].data[li] += Number.isFinite(v) ? v : 0;
        else series[si].data[li] = v;
      });
      return { labels: labels, series: series };
    }
    var ys = (spec.y || []).map(function (name) { return { name: name, i: colIndex(columns, name) }; }).filter(function (s) { return s.i >= 0; });
    return {
      labels: (rows || []).map(function (r) { return xi >= 0 ? (r[xi] == null ? '' : r[xi]) : ''; }),
      series: ys.map(function (s) {
        return { name: s.name, data: (rows || []).map(function (r) { return toNum(r[s.i]); }) };
      })
    };
  }
  function capCategories(packed, spec) {
    var max = spec.max_categories || 0;
    if (!max || packed.labels.length <= max) return packed;
    var keep = max - 1;
    return {
      labels: packed.labels.slice(0, keep).concat(['其他']),
      series: packed.series.map(function (s) {
        var extra = 0;
        for (var i = keep; i < s.data.length; i++) if (Number.isFinite(s.data[i])) extra += s.data[i];
        return { name: s.name, data: s.data.slice(0, keep).concat([extra]) };
      })
    };
  }
  function capSlices(packed, spec) {
    var s = packed.series[0];
    if (!s) return packed;
    var max = isNum(spec.max_categories) ? Number(spec.max_categories) : 8;
    var thr = isNum(spec.slice_threshold) ? Number(spec.slice_threshold) : 0;
    var total = s.data.reduce(function (a, v) { return a + (Number.isFinite(v) ? v : 0); }, 0);
    var items = packed.labels.map(function (l, i) { return { name: l, value: s.data[i] }; });
    items.sort(function (a, b) { return (Number(b.value) || 0) - (Number(a.value) || 0); });
    var keep = [], other = 0;
    var budget = max ? Math.max(max - 1, 1) : items.length;
    items.forEach(function (it, idx) {
      var pct = total ? (Number(it.value) || 0) / total * 100 : 0;
      var last = idx === items.length - 1;
      var overMax = max && keep.length >= budget && !last;
      var underThr = thr && pct < thr && keep.length;
      if (overMax || underThr) other += Number.isFinite(it.value) ? it.value : 0;
      else keep.push(it);
    });
    if (other) keep.push({ name: '其他', value: other });
    return { labels: keep.map(function (k) { return k.name; }), series: [{ name: s.name, data: keep.map(function (k) { return k.value; }) }] };
  }
  function toPercent(packed) {
    var totals = packed.labels.map(function (_, i) {
      return packed.series.reduce(function (a, s) { return a + (Number.isFinite(s.data[i]) ? s.data[i] : 0); }, 0);
    });
    return {
      labels: packed.labels,
      series: packed.series.map(function (s) {
        return {
          name: s.name,
          data: s.data.map(function (v, i) { return totals[i] ? +(v / totals[i] * 100).toFixed(2) : 0; })
        };
      })
    };
  }
  function applyMissing(data, mode) {
    return data.map(function (v) {
      if (Number.isFinite(v)) return v;
      return mode === 'zero' ? 0 : null;
    });
  }
  function linearTrend(data) {
    var pts = [];
    data.forEach(function (v, i) { if (Number.isFinite(v)) pts.push([i, v]); });
    if (pts.length < 2) return null;
    var n = pts.length, sx = 0, sy = 0, sxx = 0, sxy = 0;
    pts.forEach(function (p) { sx += p[0]; sy += p[1]; sxx += p[0] * p[0]; sxy += p[0] * p[1]; });
    var den = n * sxx - sx * sx;
    if (!den) return null;
    var b = (n * sxy - sx * sy) / den;
    var a = (sy - b * sx) / n;
    return data.map(function (_, i) { return a + b * i; });
  }
  function needsSplit(series) {
    if (series.length < 2) return false;
    var maxes = series.map(function (s) {
      return s.data.reduce(function (m, v) { return Number.isFinite(v) && Math.abs(v) > m ? Math.abs(v) : m; }, 0);
    }).filter(function (v) { return v > 0; });
    if (maxes.length < 2) return false;
    var hi = Math.max.apply(null, maxes), lo = Math.min.apply(null, maxes);
    return lo && hi / lo >= 8;
  }
  function xScaleOf(spec, labels) {
    var scale = axisSpec(spec, 'x_axis').scale || '';
    if (scale) return scale;
    var dates = (labels || []).filter(looksDate).length;
    if (dates && dates >= labels.length * 0.6) return 'timeseries';
    return 'ordinal';
  }
  function sortPackedByScale(packed, scale) {
    if (scale !== 'timeseries' && scale !== 'linear') return packed;
    var order = (packed.labels || []).map(function (label, index) {
      var value = scale === 'timeseries' ? new Date(label).getTime() : Number(label);
      return { index: index, value: Number.isFinite(value) ? value : Infinity };
    });
    order.sort(function (a, b) {
      if (a.value === b.value) return a.index - b.index;
      return a.value - b.value;
    });
    return {
      labels: order.map(function (item) { return packed.labels[item.index]; }),
      series: packed.series.map(function (series) {
        return Object.assign({}, series, {
          data: order.map(function (item) { return series.data[item.index]; })
        });
      })
    };
  }

  function disposeHost(host) {
    var el = host && host.querySelector && host.querySelector('.viz-chart');
    if (!el) return;
    if (el._tbRo) { el._tbRo.disconnect(); el._tbRo = null; }
    if (el._tbChart) { el._tbChart.dispose(); el._tbChart = null; }
  }

  function valueAxis(spec, stacking, compact, isRight) {
    var ya = axisSpec(spec, 'y_axis');
    var scale = ya.scale || 'linear';
    var hidden = ya.enabled === 'hide' || ya.enabled === false || ya.enabled === 'false';
    var auto = ya.auto_range !== false;
    var axis = {
      type: scale === 'log' ? 'log' : 'value',
      name: ya.labels === false ? '' : (ya.title || spec.y_title || ''),
      nameTextStyle: { color: '#888', fontSize: 11 },
      splitLine: hidden ? { show: false } : { lineStyle: { color: '#f0f0f0' } },
      axisTick: { show: !hidden },
      axisLine: { show: !hidden, lineStyle: { color: '#e6e6e6' } },
      axisLabel: {
        show: !hidden,
        color: '#888',
        fontSize: compact ? 10 : 11,
        formatter: function (v) {
          if (stacking === 'percent') return formatNumber(v, spec, { percent: true, alreadyPercent: true });
          return formatNumber(v, spec);
        }
      },
      scale: !!ya.unpin_zero
    };
    if (!auto) {
      if (isNum(ya.min)) axis.min = Number(ya.min);
      if (isNum(ya.max)) axis.max = Number(ya.max);
    }
    if (isNum(ya.ticks) && Number(ya.ticks) > 0) axis.splitNumber = Number(ya.ticks);
    if (isRight) axis.name = ya.labels === false ? '' : (ya.title ? '' : '');
    return axis;
  }
  function categoryAxis(spec, labels, compact, isRow) {
    var xa = axisSpec(spec, 'x_axis');
    var enabled = xa.enabled == null || xa.enabled === '' ? 'show' : String(xa.enabled);
    var hidden = enabled === 'hide' || enabled === 'false';
    var rotate = enabled === 'rotate-45' ? 45 : enabled === 'rotate-90' ? 90 : 0;
    var scale = xScaleOf(spec, labels);
    var title = xa.labels === false ? '' : (xa.title || spec.x_title || '');
    if (scale === 'timeseries') {
      return {
        type: 'time',
        name: isRow ? (axisSpec(spec, 'y_axis').title || spec.y_title || '') : title,
        nameTextStyle: { color: '#888', fontSize: 11 },
        axisLabel: { show: !hidden, color: '#666', fontSize: compact ? 10 : 11, hideOverlap: true, rotate: rotate },
        axisTick: { show: !hidden && enabled !== 'compact' },
        axisLine: { show: !hidden, lineStyle: { color: '#e6e6e6' } },
        splitLine: { show: false }
      };
    }
    if (scale === 'linear') {
      return {
        type: 'value',
        name: title,
        nameTextStyle: { color: '#888', fontSize: 11 },
        axisLabel: { show: !hidden, color: '#666', fontSize: compact ? 10 : 11, rotate: rotate },
        axisTick: { show: !hidden },
        axisLine: { show: !hidden, lineStyle: { color: '#e6e6e6' } },
        splitLine: { lineStyle: { color: '#f6f6f6' } }
      };
    }
    return {
      type: 'category',
      data: labels.map(function (v) { return v == null ? '' : String(v); }),
      name: isRow ? (axisSpec(spec, 'y_axis').title || spec.y_title || '') : title,
      nameTextStyle: { color: '#888', fontSize: 11 },
      axisLabel: {
        show: !hidden,
        color: '#666',
        fontSize: compact ? 10 : 11,
        hideOverlap: enabled === 'compact' || enabled === 'show',
        rotate: rotate,
        interval: enabled === 'compact' ? 'auto' : 0
      },
      axisTick: { show: !hidden && enabled !== 'compact' },
      axisLine: { show: !hidden, lineStyle: { color: '#e6e6e6' } },
      boundaryGap: spec.type === 'bar' || spec.type === 'row' || scale === 'histogram'
    };
  }
  function legendOption(spec, n, compact, pie) {
    if (!legendOn(spec, n)) return { show: false };
    var pos = spec.legend || (pie ? 'right' : 'top');
    var opt = { type: 'scroll', textStyle: { fontSize: compact ? 11 : 12 } };
    if (pos === 'right') Object.assign(opt, { orient: 'vertical', right: 8, top: 16 });
    else if (pos === 'left') Object.assign(opt, { orient: 'vertical', left: 8, top: 16 });
    else if (pos === 'bottom') Object.assign(opt, { bottom: 4 });
    else Object.assign(opt, { top: 4 });
    return opt;
  }

  function chartOption(spec, packed, compact) {
    var xScale = xScaleOf(spec, packed.labels);
    if (spec.type !== 'pie' && spec.type !== 'row') packed = sortPackedByScale(packed, xScale);
    var stacking = stackingOf(spec);
    if (stacking === 'percent') packed = toPercent(packed);
    var visibleSeries = packed.series.filter(function (s) { return visibleOf(spec, s.name); });
    var isRow = spec.type === 'row';
    var useTime = xScale === 'timeseries' && packed.labels.some(looksDate);
    var useLinearX = xScale === 'linear' && packed.labels.every(function (v) { return v === '' || isNum(v); });
    var split = spec.auto_split !== false && needsSplit(visibleSeries) && !stacking;
    var hasRight = visibleSeries.some(function (s, i) {
      var ax = seriesStyle(spec, s.name).axis;
      if (ax === 'right') return true;
      if (ax === 'left') return false;
      return split && i === visibleSeries.length - 1;
    });

    function pointData(s, st) {
      var raw = applyMissing(s.data, missingOf(spec, st));
      var scale = axisSpec(spec, 'y_axis').scale;
      if (scale === 'pow' && !stacking) {
        raw = raw.map(function (v) { return v == null ? null : Math.sign(v) * Math.sqrt(Math.abs(v)); });
      }
      if (useTime) {
        return packed.labels.map(function (l, i) {
          var t = looksDate(l) ? new Date(l).getTime() : i;
          return [t, raw[i]];
        });
      }
      if (useLinearX) {
        return packed.labels.map(function (l, i) { return [Number(l), raw[i]]; });
      }
      return raw;
    }

    if (spec.type === 'pie') {
      var pieData = packed.labels.map(function (label, i) {
        var v = packed.series[0] ? packed.series[0].data[i] : 0;
        var name = String(label ?? '');
        return {
          name: titleOf(spec, name) || name,
          value: Number.isFinite(v) ? v : 0,
          itemStyle: { color: colorOf(spec, name, i) }
        };
      }).filter(function (d, i) {
        var rawName = packed.labels[i];
        return visibleOf(spec, rawName);
      });
      var total = pieData.reduce(function (a, d) { return a + d.value; }, 0);
      var pct = percentMode(spec);
      var showLabels = spec.show_labels || pct === 'inside' || pct === 'both';
      var showLegend = legendOn(spec, pieData.length);
      var legendPos = spec.legend || 'right';
      var donut = spec.donut !== false;
      var showTotal = spec.show_total !== false;
      var legend = legendOption(spec, pieData.length, compact, true);
      if (showLegend && (pct === 'legend' || pct === 'both')) {
        legend.formatter = function (name) {
          var item = pieData.find(function (d) { return d.name === name; });
          var p = total ? ((item ? item.value : 0) / total * 100) : 0;
          var dec = spec.decimals == null ? 0 : Number(spec.decimals);
          return name + '  ' + p.toFixed(Number.isFinite(dec) ? dec : 0) + '%';
        };
      }
      var labelFmt = function (p) {
        var bits = [];
        if (spec.show_labels) bits.push(p.name);
        if (pct === 'inside' || pct === 'both') bits.push(p.percent.toFixed(spec.decimals == null ? 0 : Number(spec.decimals)) + '%');
        return bits.join('\n') || p.name;
      };
      var center = showLegend && (legendPos === 'right' || legendPos === 'left') ? ['38%', '52%'] : ['50%', '50%'];
      if (legendPos === 'left') center = ['62%', '52%'];
      var option = {
        color: COLORS,
        animationDuration: 700,
        animationEasing: 'cubicOut',
        tooltip: { trigger: 'item', formatter: function (p) { return esc(p.name) + '<br/>' + formatNumber(p.value, spec) + '（' + p.percent.toFixed(1) + '%）'; } },
        legend: legend,
        series: [{
          type: 'pie',
          radius: donut ? (compact ? ['42%', '68%'] : ['46%', '72%']) : (compact ? '68%' : '72%'),
          center: center,
          itemStyle: { borderColor: '#fff', borderWidth: 2, borderRadius: donut ? 6 : 4 },
          label: { show: showLabels, formatter: labelFmt, fontSize: 11 },
          labelLine: { show: showLabels && !donut },
          data: pieData
        }]
      };
      if (donut && showTotal) {
        option.graphic = [{
          type: 'text',
          left: center[0],
          top: 'middle',
          bounding: 'raw',
          style: {
            text: formatNumber(total, spec) + '\n总计',
            textAlign: 'center',
            textVerticalAlign: 'middle',
            fill: '#222',
            fontSize: compact ? 13 : 16,
            fontWeight: 650,
            lineHeight: compact ? 18 : 22
          }
        }];
      }
      return option;
    }

    var series = visibleSeries.map(function (s, si) {
      var st = seriesStyle(spec, s.name);
      var kind = isRow ? 'bar' : displayOf(spec, st);
      var interp = interpOf(spec, st);
      var ax = st.axis === 'right' || (!st.axis && split && si === visibleSeries.length - 1) ? 'right' : 'left';
      var item = {
        name: titleOf(spec, s.name),
        type: kind === 'bar' ? 'bar' : (kind === 'scatter' ? 'scatter' : 'line'),
        data: pointData(s, st),
        yAxisIndex: hasRight && ax === 'right' && !isRow ? 1 : 0,
        showSymbol: kind === 'bar' ? false : (kind === 'scatter' || showDots(spec, st, packed.labels.length)),
        smooth: kind !== 'bar' && kind !== 'scatter' && interp === 'cardinal',
        step: interp === 'step' ? 'end' : false,
        connectNulls: missingOf(spec, st) === 'interpolate',
        symbolSize: 7,
        emphasis: { focus: 'series' },
        itemStyle: { color: colorOf(spec, s.name, si) },
        label: (spec.show_labels && st.show_values !== false) ? {
          show: spec.label_frequency !== 'fit',
          position: isRow ? 'right' : 'top',
          fontSize: 10,
          formatter: function (p) {
            var v = Array.isArray(p.data) ? p.data[1] : p.data;
            if (axisSpec(spec, 'y_axis').scale === 'pow' && !stacking && Number.isFinite(v)) v = Math.sign(v) * v * v;
            return stacking === 'percent' ? formatNumber(v, spec, { percent: true, alreadyPercent: true }) : formatNumber(v, spec);
          }
        } : undefined
      };
      if (spec.label_frequency === 'fit' && spec.show_labels) {
        item.label = Object.assign({}, item.label, { show: true, hideOverlap: true });
      }
      if (stacking) item.stack = 'total';
      if (kind === 'bar') {
        item.barMaxWidth = compact ? 22 : 36;
        item.itemStyle.borderRadius = isRow ? [0, 6, 6, 0] : [6, 6, 0, 0];
      }
      if (kind === 'area' || spec.type === 'area' && kind !== 'bar') {
        item.areaStyle = { opacity: stacking ? 0.85 : 0.18, color: colorOf(spec, s.name, si) };
        item.lineStyle = { width: lineWidth(st), type: st.line_style || 'solid', color: colorOf(spec, s.name, si) };
      }
      if (kind === 'line') {
        item.lineStyle = { width: lineWidth(st), type: st.line_style || 'solid', color: colorOf(spec, s.name, si) };
      }
      if (goalOn(spec) && si === 0 && stacking !== 'percent') {
        var g = Number(spec.goal);
        if (axisSpec(spec, 'y_axis').scale === 'pow') g = Math.sign(g) * Math.sqrt(Math.abs(g));
        item.markLine = {
          symbol: 'none',
          label: { formatter: spec.goal_label || ('目标 ' + formatNumber(Number(spec.goal), spec)), color: '#EF8C8C' },
          lineStyle: { color: '#EF8C8C', type: 'dashed' },
          data: [isRow ? { xAxis: g } : { yAxis: g }]
        };
      }
      return item;
    });

    visibleSeries.forEach(function (s, si) {
      var st = seriesStyle(spec, s.name);
      var on = spec.trendline && (visibleSeries.length === 1 || st.show_trend !== false);
      if (!on) return;
      var trend = linearTrend(applyMissing(s.data, missingOf(spec, st)));
      if (!trend) return;
      var fake = { name: s.name + ' · 趋势', data: trend };
      var st2 = { missing: 'interpolate', markers: 'off', line_style: 'dashed', line_size: 'S' };
      series.push({
        name: titleOf(spec, s.name) + ' · 趋势',
        type: 'line',
        data: pointData(fake, st2),
        yAxisIndex: series[si] ? series[si].yAxisIndex : 0,
        showSymbol: false,
        silent: true,
        lineStyle: { width: 1.6, type: 'dashed', color: colorOf(spec, s.name, si) },
        itemStyle: { color: colorOf(spec, s.name, si) }
      });
    });

    var catAxis = categoryAxis(spec, packed.labels, compact, isRow);
    var valAxis = valueAxis(spec, stacking, compact, false);
    var yAxes = hasRight && !isRow ? [valAxis, Object.assign({}, valueAxis(spec, stacking, compact, true), { splitLine: { show: false } })] : valAxis;
    var showL = legendOn(spec, visibleSeries.length);
    var lPos = spec.legend || 'top';
    var grid = {
      left: isRow ? 12 : (lPos === 'left' ? 72 : 16),
      right: 18 + (hasRight ? 12 : 0) + (lPos === 'right' ? 70 : 0),
      top: showL && (lPos === 'top' || !lPos) ? 36 : 18,
      bottom: showL && lPos === 'bottom' ? 36 : 12,
      containLabel: true
    };
    return {
      color: visibleSeries.map(function (s, i) { return colorOf(spec, s.name, i); }),
      animationDuration: 700,
      animationEasing: 'cubicOut',
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: spec.type === 'bar' || spec.type === 'row' ? 'shadow' : 'line' },
        valueFormatter: function (v) {
          if (axisSpec(spec, 'y_axis').scale === 'pow' && stacking !== 'percent' && Number.isFinite(v)) v = Math.sign(v) * v * v;
          return stacking === 'percent' ? formatNumber(v, spec, { percent: true, alreadyPercent: true }) : formatNumber(v, spec);
        }
      },
      legend: legendOption(spec, visibleSeries.length, compact, false),
      grid: grid,
      xAxis: isRow ? valAxis : catAxis,
      yAxis: isRow ? catAxis : yAxes,
      series: series
    };
  }

  function mountChart(host, option, spec) {
    disposeHost(host);
    host.innerHTML = '<div class="viz-chart"></div>';
    var el = host.querySelector('.viz-chart');
    if (!global.echarts) {
      host.innerHTML = '<div class="viz-empty"><b>无法绘制图表</b><p>图表组件没有加载成功，请刷新页面。</p></div>';
      return;
    }
    var chart = global.echarts.init(el, spec.dashboard_theme && spec.dashboard_theme !== 'light' ? 'dark' : null, { renderer: 'canvas' });
    chart.setOption(option, true);
    el._tbChart = chart;
    el._tbRo = new ResizeObserver(function () { chart.resize(); });
    el._tbRo.observe(el);
  }
  function renderTypes(host, spec, onPick) {
    host.innerHTML = TYPES.map(function (t) {
      return '<button type="button" data-type="' + t.id + '" class="' + (spec.type === t.id ? 'active' : '') + '">' + t.label + '</button>';
    }).join('');
    host.querySelectorAll('[data-type]').forEach(function (btn) {
      btn.onclick = function () { onPick(btn.dataset.type); };
    });
  }
  function seg(name, value, options, seriesName) {
    var extra = seriesName ? ' data-series="' + esc(seriesName) + '"' : '';
    return '<div class="viz-seg">' + options.map(function (opt) {
      return '<button type="button" data-k="' + name + '" data-v="' + esc(String(opt.value)) + '"' + extra + ' class="' + (String(value || '') === String(opt.value) ? 'active' : '') + '">' + esc(opt.label) + '</button>';
    }).join('') + '</div>';
  }
  function field(label, control) {
    return '<label class="viz-field"><span>' + esc(label) + '</span>' + control + '</label>';
  }
  function unusedNames(all, taken, keep) {
    return (all || []).filter(function (name) {
      return name === keep || taken.indexOf(name) < 0;
    });
  }
  function axisPickers(label, all, selected, key, opts) {
    opts = opts || {};
    var values = (selected || []).filter(function (v, i) { return v || i === 0; });
    if (!values.length) values = [all[0] || ''];
    var html = '<div class="viz-field"><span>' + esc(label) + '</span>';
    values.forEach(function (val, i) {
      var options = unusedNames(all, values, val);
      if (val && options.indexOf(val) < 0) options = [val].concat(options);
      var canRemove = opts.canRemove ? opts.canRemove(i, values) : values.length > 1;
      html += '<div class="viz-axis-row">';
      html += '<select data-pick="' + key + '" data-i="' + i + '">' + optionList(options, val) + '</select>';
      if (canRemove) html += '<button type="button" class="viz-icon" data-pick-del="' + key + '" data-i="' + i + '" title="移除">×</button>';
      html += '</div>';
    });
    if (opts.addLabel && opts.canAdd) {
      html += '<button type="button" class="viz-add-link" data-pick-add="' + key + '">+ ' + esc(opts.addLabel) + '</button>';
    }
    html += '</div>';
    return html;
  }
  function optionList(values, selected, placeholder) {
    return (placeholder != null ? '<option value="">' + esc(placeholder) + '</option>' : '') +
      values.map(function (c) {
        var v = typeof c === 'string' ? c : c.value;
        var l = typeof c === 'string' ? c : c.label;
        return '<option value="' + esc(v) + '"' + (String(selected) === String(v) ? ' selected' : '') + '>' + esc(l) + '</option>';
      }).join('');
  }
  function swatches(name, current) {
    return '<div class="viz-swatches">' + COLORS.map(function (c) {
      return '<button type="button" class="viz-swatch' + (String(current || '').toLowerCase() === c.toLowerCase() ? ' active' : '') +
        '" data-series="' + esc(name) + '" data-color="' + c + '" style="background:' + c + '" title="' + c + '"></button>';
    }).join('') + '</div>';
  }
  function tabsFor(spec) {
    if (spec.type === 'table') return [{ id: 'display', label: '显示' }];
    if (spec.type === 'scalar') return [{ id: 'data', label: '数据' }, { id: 'format', label: '格式' }, { id: 'color', label: '颜色' }];
    if (spec.type === 'pie') return [{ id: 'data', label: '数据' }, { id: 'display', label: '显示' }];
    return [{ id: 'data', label: '数据' }, { id: 'display', label: '显示' }, { id: 'axes', label: '坐标轴' }];
  }
  function tabsHtml(spec, tab) {
    var tabs = tabsFor(spec);
    return '<div class="tb-tabs viz-tabs">' + tabs.map(function (t) {
      return '<button type="button" data-tab="' + t.id + '" class="' + (t.id === tab ? 'active' : '') + '">' + t.label + '</button>';
    }).join('') + '</div>';
  }
  function seriesCard(spec, name, index, cartesian, open) {
    var st = seriesStyle(spec, name);
    var color = colorOf(spec, name, index);
    var kind = displayOf(spec, st);
    var html = '<details class="viz-series-card" data-series="' + esc(name) + '"' + (open ? ' open' : '') + '>';
    html += '<summary><input type="color" data-series="' + esc(name) + '" data-sk="color" value="' + esc(color) + '">' +
      '<input type="text" data-series="' + esc(name) + '" data-sk="title" value="' + esc(st.title || name) + '" placeholder="' + esc(name) + '">' +
      '<label class="viz-mini"><input type="checkbox" data-series="' + esc(name) + '" data-sk="visible"' + (st.visible !== false ? ' checked' : '') + '> 显示</label></summary>';
    html += swatches(name, color);
    if (cartesian && spec.type !== 'row') {
      html += field('展示类型', seg('display', kind, [{ value: 'line', label: '折线' }, { value: 'area', label: '面积' }, { value: 'bar', label: '柱状' }], name));
      html += field('Y 轴位置', seg('axis', st.axis || '', [{ value: '', label: '自动' }, { value: 'left', label: '左' }, { value: 'right', label: '右' }], name));
    }
    if (cartesian && (kind === 'line' || kind === 'area' || spec.type === 'line' || spec.type === 'area')) {
      html += field('折线形状', seg('interpolate', interpOf(spec, st), [{ value: 'linear', label: '直线' }, { value: 'cardinal', label: '平滑' }, { value: 'step', label: '阶梯' }], name));
      html += field('线型', seg('line_style', st.line_style || 'solid', [{ value: 'solid', label: '实线' }, { value: 'dashed', label: '虚线' }, { value: 'dotted', label: '点线' }], name));
      html += field('粗细', seg('line_size', String(st.line_size || 'M').toUpperCase(), [{ value: 'S', label: 'S' }, { value: 'M', label: 'M' }, { value: 'L', label: 'L' }], name));
      html += field('空值', seg('missing', missingOf(spec, st), [{ value: 'interpolate', label: '插值' }, { value: 'zero', label: '记为 0' }, { value: 'none', label: '断开' }], name));
      html += field('显示圆点', seg('markers', st.markers || spec.markers || 'auto', [{ value: 'auto', label: '自动' }, { value: 'on', label: '开' }, { value: 'off', label: '关' }], name));
    }
    html += '</details>';
    return html;
  }

  function renderSettings(host, columns, rows, spec, onChange) {
    if (!host) return;
    var kind = classify(columns, rows);
    var dims = kind.dims.length ? kind.dims : columns;
    var metrics = kind.metrics.length ? kind.metrics : columns;
    var chartLike = spec.type !== 'table' && spec.type !== 'scalar';
    var cartesian = spec.type === 'line' || spec.type === 'area' || spec.type === 'bar' || spec.type === 'row';
    var stacking = stackingOf(spec);
    var packed = columns.length ? seriesValues(columns, rows, spec) : { labels: [], series: [] };
    if (spec.type === 'pie') packed = capSlices(packed, spec);
    var allowed = tabsFor(spec).map(function (t) { return t.id; });
    var tab = host.dataset.tab || allowed[0];
    if (allowed.indexOf(tab) < 0) tab = allowed[0];
    host.dataset.tab = tab;
    var openSeries = (host.dataset.openSeries || '').split(',').filter(Boolean);
    var html = '<h3>设置</h3>';
    if (!columns.length) {
      host.innerHTML = html + '<p class="viz-muted">查询有结果后，可以在这里选择图表类型、颜色和坐标轴。</p>';
      return;
    }
    html += tabsHtml(spec, tab);

    if (spec.type === 'table' && tab === 'display') {
      html += '<p class="viz-muted">设置字段的显示状态和名称。数据筛选请使用页面顶部的筛选器。</p>';
      html += (columns || []).map(function (c) {
        var cs = columnStyle(spec, c);
        return '<div class="viz-col-block"><div class="viz-col-row"><label class="viz-mini"><input type="checkbox" data-col="' + esc(c) + '" data-ck="visible"' + (cs.visible !== false ? ' checked' : '') + '></label>' +
          '<input type="text" data-col="' + esc(c) + '" data-ck="title" value="' + esc(cs.title || '') + '" placeholder="' + esc(c) + '"></div></div>';
      }).join('');
    }

    if (tab === 'data' && spec.type === 'scalar') {
      html += field('要显示的字段', '<select data-k="scalar">' + optionList(metrics.length ? metrics : columns, (spec.y || [])[0]) + '</select>');
    }
    if (tab === 'data' && spec.type === 'pie') {
      html += field('维度', '<select data-k="x">' + optionList(dims, spec.x) + '</select>');
      html += field('指标', '<select data-k="scalar">' + optionList(metrics, (spec.y || [])[0]) + '</select>');
      html += '<h4>切片颜色</h4>';
      html += packed.labels.map(function (name, i) { return seriesCard(spec, name, i, false, openSeries.indexOf(name) >= 0 || (!openSeries.length && i === 0)); }).join('');
      html += field('「其他」颜色', '<input type="color" data-k="other_color" value="' + esc(spec.other_color || '#7172AD') + '">');
    }
    if (tab === 'data' && cartesian) {
      var xVals = spec.breakout ? [spec.x, spec.breakout] : [spec.x || dims[0] || ''];
      var yVals = (spec.y && spec.y.length ? spec.y.slice() : (metrics[0] ? [metrics[0]] : []));
      var unusedDims = dims.filter(function (d) { return d && d !== spec.x && d !== spec.breakout; });
      var unusedMetrics = metrics.filter(function (m) { return yVals.indexOf(m) < 0; });
      html += axisPickers('横轴', dims, xVals, 'x', {
        addLabel: '添加分组',
        canAdd: !spec.breakout && yVals.length <= 1 && unusedDims.length > 0,
        canRemove: function (i) { return i > 0; }
      });
      html += axisPickers('纵轴', metrics.length ? metrics : columns, yVals, 'y', {
        addLabel: '添加指标',
        canAdd: !spec.breakout && unusedMetrics.length > 0,
        canRemove: function (i, values) { return values.length > 1; }
      });
      html += '<h4>系列</h4>';
      html += packed.series.map(function (s, i) { return seriesCard(spec, s.name, i, true, openSeries.indexOf(s.name) >= 0 || (!openSeries.length && i === 0)); }).join('');
    }

    if (tab === 'display' && cartesian) {
      if (spec.type === 'bar' || spec.type === 'area' || spec.type === 'row' || spec.type === 'line') {
        html += field('堆叠', seg('stacking', stacking, [{ value: '', label: '不堆叠' }, { value: 'stack', label: '堆叠' }, { value: 'percent', label: '百分比' }]));
      }
      html += '<label class="viz-toggle"><input type="checkbox" data-k="legend_on"' + (legendOn(spec, 1) ? ' checked' : '') + '> 显示图例</label>';
      if (legendOn(spec, 1)) {
        html += field('图例位置', seg('legend', spec.legend || 'top', [{ value: 'top', label: '上' }, { value: 'bottom', label: '下' }, { value: 'left', label: '左' }, { value: 'right', label: '右' }]));
      }
      html += '<label class="viz-toggle"><input type="checkbox" data-k="labels"' + (spec.show_labels ? ' checked' : '') + '> 显示数值</label>';
      if (spec.show_labels) {
        html += field('显示哪些数值', seg('label_frequency', spec.label_frequency || 'all', [{ value: 'fit', label: '部分' }, { value: 'all', label: '全部' }]));
        html += field('数值格式', seg('value_format', spec.value_format || 'auto', [{ value: 'auto', label: '自动' }, { value: 'compact', label: '紧凑' }, { value: 'full', label: '完整' }]));
      }
      html += '<label class="viz-toggle"><input type="checkbox" data-k="trendline"' + (spec.trendline ? ' checked' : '') + '> 趋势线</label>';
      html += '<label class="viz-toggle"><input type="checkbox" data-k="show_goal"' + (goalOn(spec) || spec.show_goal === true ? ' checked' : '') + '> 目标线</label>';
      if (goalOn(spec) || spec.show_goal === true) {
        html += field('目标值', '<input data-k="goal" type="number" placeholder="例如 100" value="' + (spec.goal == null ? '' : spec.goal) + '">');
        html += field('目标标签', '<input data-k="goal_label" type="text" placeholder="目标" value="' + esc(spec.goal_label || '') + '">');
      }
      if (spec.type === 'bar' || spec.type === 'line' || spec.type === 'area') {
        html += '<label class="viz-toggle"><input type="checkbox" data-k="auto_split"' + (spec.auto_split !== false ? ' checked' : '') + '> 量级差过大时拆分纵轴</label>';
      }
    }

    if (tab === 'display' && spec.type === 'pie') {
      html += '<label class="viz-toggle"><input type="checkbox" data-k="legend_on"' + (legendOn(spec, 1) ? ' checked' : '') + '> 显示图例</label>';
      if (legendOn(spec, 1)) {
        html += field('图例位置', seg('legend', spec.legend || 'right', [{ value: 'top', label: '上' }, { value: 'bottom', label: '下' }, { value: 'left', label: '左' }, { value: 'right', label: '右' }]));
      }
      html += '<label class="viz-toggle"><input type="checkbox" data-k="show_total"' + (spec.show_total !== false ? ' checked' : '') + '> 显示总计</label>';
      html += '<label class="viz-toggle"><input type="checkbox" data-k="donut"' + (spec.donut !== false ? ' checked' : '') + '> 环形图</label>';
      html += '<label class="viz-toggle"><input type="checkbox" data-k="labels"' + (spec.show_labels ? ' checked' : '') + '> 显示标签</label>';
      html += field('显示百分比', seg('percent', percentMode(spec), [
        { value: 'off', label: '关闭' }, { value: 'legend', label: '图例' }, { value: 'inside', label: '图上' }, { value: 'both', label: '两者' }
      ]));
      if (percentMode(spec) !== 'off') {
        html += field('百分比小数位', '<input data-k="decimals" type="number" min="0" max="6" placeholder="自动" value="' + (spec.decimals == null ? '' : spec.decimals) + '">');
      }
      html += field('最多分类', '<input data-k="max" type="number" min="2" max="40" value="' + (spec.max_categories || 8) + '">');
      html += field('最小切片占比 %', '<input data-k="slice_threshold" type="number" min="0" max="50" step="0.5" value="' + (spec.slice_threshold == null ? '' : spec.slice_threshold) + '" placeholder="0">');
    }

    if (tab === 'axes' && cartesian) {
      var xa = axisSpec(spec, 'x_axis');
      var ya = axisSpec(spec, 'y_axis');
      html += '<div class="viz-axis-group"><strong>横轴</strong>';
      html += '<label class="viz-toggle"><input type="checkbox" data-axis="x_axis" data-ak="labels"' + (xa.labels !== false ? ' checked' : '') + '> 显示轴标题</label>';
      if (xa.labels !== false) html += field('轴标题', '<input data-axis="x_axis" data-ak="title" type="text" placeholder="默认使用字段名" value="' + esc(xa.title || spec.x_title || '') + '">');
      html += field('刻度', '<select data-axis="x_axis" data-ak="enabled">' + optionList(
        [{ value: 'show', label: '显示' }, { value: 'compact', label: '紧凑' }, { value: 'rotate-45', label: '旋转 45°' }, { value: 'rotate-90', label: '旋转 90°' }, { value: 'hide', label: '隐藏' }],
        xa.enabled == null || xa.enabled === '' || xa.enabled === true || xa.enabled === 'true' ? 'show' : String(xa.enabled)
      ) + '</select>');
      html += field('刻度类型', '<select data-axis="x_axis" data-ak="scale">' + optionList(
        [{ value: 'ordinal', label: '分类' }, { value: 'timeseries', label: '时间' }, { value: 'linear', label: '线性' }, { value: 'histogram', label: '直方图' }],
        xScaleOf(spec, packed.labels)
      ) + '</select>');
      html += '</div><div class="viz-axis-group"><strong>纵轴</strong>';
      html += '<label class="viz-toggle"><input type="checkbox" data-axis="y_axis" data-ak="labels"' + (ya.labels !== false ? ' checked' : '') + '> 显示轴标题</label>';
      if (ya.labels !== false) html += field('轴标题', '<input data-axis="y_axis" data-ak="title" type="text" placeholder="默认使用字段名" value="' + esc(ya.title || spec.y_title || '') + '">');
      html += field('刻度', '<select data-axis="y_axis" data-ak="enabled">' + optionList([{ value: 'show', label: '显示' }, { value: 'hide', label: '隐藏' }], ya.enabled === 'hide' || ya.enabled === false ? 'hide' : 'show') + '</select>');
      html += field('刻度类型', '<select data-axis="y_axis" data-ak="scale">' + optionList([{ value: 'linear', label: '线性' }, { value: 'log', label: '对数' }, { value: 'pow', label: '幂次' }], ya.scale || 'linear') + '</select>');
      html += '<label class="viz-toggle"><input type="checkbox" data-axis="y_axis" data-ak="auto_range"' + (ya.auto_range !== false ? ' checked' : '') + '> 自动范围</label>';
      if (ya.auto_range === false) {
        html += '<div class="viz-range">' + field('最小值', '<input data-axis="y_axis" data-ak="min" type="number" value="' + (ya.min == null ? '' : ya.min) + '">') +
          field('最大值', '<input data-axis="y_axis" data-ak="max" type="number" value="' + (ya.max == null ? '' : ya.max) + '">') + '</div>';
      }
      html += '<label class="viz-toggle"><input type="checkbox" data-axis="y_axis" data-ak="unpin_zero"' + (ya.unpin_zero ? ' checked' : '') + '> 脱离 0 点</label>';
      html += field('刻度数量', '<input data-axis="y_axis" data-ak="ticks" type="number" min="2" max="20" placeholder="自动" value="' + (ya.ticks == null ? '' : ya.ticks) + '">');
      html += '</div>';
    }

    if (tab === 'format' && spec.type === 'scalar') {
      html += field('样式', seg('number_style', spec.number_style || 'number', [{ value: 'number', label: '数字' }, { value: 'percent', label: '百分比' }, { value: 'currency', label: '货币' }, { value: 'scientific', label: '科学计数' }]));
      html += field('小数位', '<input data-k="decimals" type="number" min="0" max="8" placeholder="自动" value="' + (spec.decimals == null ? '' : spec.decimals) + '">');
      html += field('前缀', '<input data-k="prefix" type="text" placeholder="例如 ¥" value="' + esc(spec.prefix || '') + '">');
      html += field('后缀', '<input data-k="suffix" type="text" placeholder="例如 单" value="' + esc(spec.suffix || '') + '">');
      html += field('乘以系数', '<input data-k="multiply" type="number" step="any" placeholder="1" value="' + (spec.multiply == null ? '' : spec.multiply) + '">');
    }
    if (tab === 'color' && spec.type === 'scalar') {
      html += field('数字颜色', '<input type="color" data-k="color" value="' + esc(spec.color || '#111111') + '">');
      html += swatches('__scalar__', spec.color || '#111111');
      html += '<h4>条件颜色</h4><p class="viz-muted">按数值范围改颜色，区间包含两端。</p>';
      (spec.segments || []).forEach(function (segRow, i) {
        html += '<div class="viz-segment-row" data-seg="' + i + '">' +
          '<input type="number" data-seg="' + i + '" data-sf="min" placeholder="最小" value="' + (segRow.min == null ? '' : segRow.min) + '">' +
          '<input type="number" data-seg="' + i + '" data-sf="max" placeholder="最大" value="' + (segRow.max == null ? '' : segRow.max) + '">' +
          '<input type="color" data-seg="' + i + '" data-sf="color" value="' + esc(segRow.color || COLORS[i % COLORS.length]) + '">' +
          '<button type="button" class="viz-icon" data-seg-del="' + i + '" title="删除">×</button></div>';
      });
      html += '<button type="button" class="viz-add" data-k="add_segment">添加区间</button>';
    }

    host.innerHTML = html;

    function emit(next) { onChange(Object.assign({}, spec, next)); }
    function emitSeries(name, patch) {
      var series = Object.assign({}, spec.series || {});
      series[name] = Object.assign({}, series[name] || {}, patch);
      emit({ series: series });
    }
    function emitAxis(which, patch) {
      var next = {};
      next[which] = Object.assign({}, spec[which] || {}, patch);
      if (which === 'x_axis' && patch.title != null) next.x_title = patch.title;
      if (which === 'y_axis' && patch.title != null) next.y_title = patch.title;
      emit(next);
    }
    function emitColumn(name, patch) {
      var cols = Object.assign({}, spec.columns || {});
      cols[name] = Object.assign({}, cols[name] || {}, patch);
      emit({ columns: cols });
    }

    host.querySelectorAll('details summary input').forEach(function (el) {
      el.onclick = function (e) { e.stopPropagation(); };
    });
    host.querySelectorAll('[data-tab]').forEach(function (btn) {
      btn.onclick = function () { host.dataset.tab = btn.dataset.tab; renderSettings(host, columns, rows, spec, onChange); };
    });
    host.querySelectorAll('details[data-series]').forEach(function (el) {
      el.ontoggle = function () {
        var names = [];
        host.querySelectorAll('details[data-series]').forEach(function (d) { if (d.open) names.push(d.dataset.series); });
        host.dataset.openSeries = names.join(',');
      };
    });
    var x = host.querySelector('[data-k="x"]');
    if (x) x.onchange = function () { emit({ x: x.value }); };
    var scalar = host.querySelector('[data-k="scalar"]');
    if (scalar) scalar.onchange = function () { emit({ y: [scalar.value] }); };
    host.querySelectorAll('select[data-pick]').forEach(function (el) {
      el.onchange = function () {
        var i = Number(el.dataset.i);
        if (el.dataset.pick === 'x') {
          if (i === 0) emit({ x: el.value, breakout: spec.breakout === el.value ? '' : spec.breakout });
          else emit({ breakout: el.value });
          return;
        }
        var y = (spec.y && spec.y.length ? spec.y.slice() : []);
        if (!y.length) y = [el.value];
        else y[i] = el.value;
        emit({ y: y.filter(Boolean) });
      };
    });
    host.querySelectorAll('[data-pick-del]').forEach(function (btn) {
      btn.onclick = function () {
        if (btn.dataset.pickDel === 'x') {
          emit({ breakout: '' });
          return;
        }
        var y = (spec.y || []).slice();
        y.splice(Number(btn.dataset.i), 1);
        emit({ y: y.length ? y : (spec.y || []).slice(0, 1) });
      };
    });
    host.querySelectorAll('[data-pick-add]').forEach(function (btn) {
      btn.onclick = function () {
        if (btn.dataset.pickAdd === 'x') {
          var dim = dims.find(function (d) { return d && d !== spec.x && d !== spec.breakout; });
          if (dim) emit({ breakout: dim, y: (spec.y || []).slice(0, 1) });
          return;
        }
        var y = (spec.y || []).slice();
        var metric = metrics.find(function (m) { return y.indexOf(m) < 0; });
        if (metric) emit({ y: y.concat([metric]), breakout: '' });
      };
    });
    host.querySelectorAll('[data-k="stacking"]').forEach(function (btn) {
      btn.onclick = function () { emit({ stacking: btn.dataset.v, stacked: btn.dataset.v === 'stack' }); };
    });
    host.querySelectorAll('.viz-seg button[data-k]:not([data-series])').forEach(function (btn) {
      if (btn.dataset.k === 'stacking') return;
      btn.onclick = function () {
        var patch = {};
        patch[btn.dataset.k] = btn.dataset.v;
        if (btn.dataset.k === 'legend') patch.show_legend = btn.dataset.v === 'off' ? false : undefined;
        emit(patch);
      };
    });
    host.querySelectorAll('.viz-seg button[data-series]').forEach(function (btn) {
      btn.onclick = function () {
        var patch = {};
        patch[btn.dataset.k] = btn.dataset.v;
        emitSeries(btn.dataset.series, patch);
      };
    });
    var legendOnEl = host.querySelector('[data-k="legend_on"]');
    if (legendOnEl) legendOnEl.onchange = function () { emit({ show_legend: legendOnEl.checked ? undefined : false, legend: legendOnEl.checked ? (spec.legend === 'off' ? 'top' : spec.legend) : 'off' }); };
    var labels = host.querySelector('[data-k="labels"]');
    if (labels) labels.onchange = function () { emit({ show_labels: labels.checked }); };
    var trend = host.querySelector('[data-k="trendline"]');
    if (trend) trend.onchange = function () { emit({ trendline: trend.checked }); };
    var showGoal = host.querySelector('[data-k="show_goal"]');
    if (showGoal) showGoal.onchange = function () { emit({ show_goal: showGoal.checked }); };
    var autoSplit = host.querySelector('[data-k="auto_split"]');
    if (autoSplit) autoSplit.onchange = function () { emit({ auto_split: autoSplit.checked ? undefined : false }); };
    var showTotal = host.querySelector('[data-k="show_total"]');
    if (showTotal) showTotal.onchange = function () { emit({ show_total: showTotal.checked ? undefined : false }); };
    var donut = host.querySelector('[data-k="donut"]');
    if (donut) donut.onchange = function () { emit({ donut: donut.checked ? undefined : false }); };
    var max = host.querySelector('[data-k="max"]');
    if (max) max.onchange = function () { emit({ max_categories: Number(max.value) || 8 }); };
    var thr = host.querySelector('[data-k="slice_threshold"]');
    if (thr) thr.onchange = function () { emit({ slice_threshold: thr.value === '' ? null : Number(thr.value) }); };
    var goal = host.querySelector('[data-k="goal"]');
    if (goal) goal.onchange = function () { emit({ goal: goal.value === '' ? null : Number(goal.value) }); };
    var gl = host.querySelector('[data-k="goal_label"]');
    if (gl) gl.onchange = function () { emit({ goal_label: gl.value }); };
    var dec = host.querySelector('[data-k="decimals"]');
    if (dec) dec.onchange = function () { emit({ decimals: dec.value === '' ? null : Number(dec.value) }); };
    var prefix = host.querySelector('[data-k="prefix"]');
    if (prefix) prefix.onchange = function () { emit({ prefix: prefix.value }); };
    var suffix = host.querySelector('[data-k="suffix"]');
    if (suffix) suffix.onchange = function () { emit({ suffix: suffix.value }); };
    var mul = host.querySelector('[data-k="multiply"]');
    if (mul) mul.onchange = function () { emit({ multiply: mul.value === '' ? null : Number(mul.value) }); };
    var colorEl = host.querySelector('[data-k="color"]');
    if (colorEl) colorEl.onchange = function () { emit({ color: colorEl.value }); };
    var other = host.querySelector('[data-k="other_color"]');
    if (other) other.onchange = function () { emit({ other_color: other.value }); };
    host.querySelectorAll('input[data-sk][data-series]').forEach(function (el) {
      el.onchange = function () {
        var patch = {};
        if (el.dataset.sk === 'visible') patch.visible = el.checked ? undefined : false;
        else if (el.dataset.sk === 'color' || el.dataset.sk === 'title') patch[el.dataset.sk] = el.value;
        emitSeries(el.dataset.series, patch);
      };
    });
    host.querySelectorAll('.viz-swatch[data-series]').forEach(function (btn) {
      btn.onclick = function () {
        if (btn.dataset.series === '__scalar__') emit({ color: btn.dataset.color });
        else emitSeries(btn.dataset.series, { color: btn.dataset.color });
      };
    });
    host.querySelectorAll('[data-axis][data-ak]').forEach(function (el) {
      el.onchange = function () {
        var patch = {};
        var key = el.dataset.ak;
        if (el.type === 'checkbox') {
          if (key === 'labels') patch.labels = el.checked ? undefined : false;
          else if (key === 'auto_range') patch.auto_range = el.checked ? undefined : false;
          else if (key === 'unpin_zero') patch.unpin_zero = el.checked;
        } else if (key === 'min' || key === 'max' || key === 'ticks') {
          patch[key] = el.value === '' ? null : Number(el.value);
        } else patch[key] = el.value;
        emitAxis(el.dataset.axis, patch);
      };
    });
    host.querySelectorAll('[data-col][data-ck]').forEach(function (el) {
      el.onchange = function () {
        var patch = {};
        if (el.dataset.ck === 'visible') patch.visible = el.checked ? undefined : false;
        else patch.title = el.value;
        emitColumn(el.dataset.col, patch);
      };
    });
    host.querySelectorAll('[data-seg][data-sf]').forEach(function (el) {
      el.onchange = function () {
        var segs = (spec.segments || []).map(function (s) { return Object.assign({}, s); });
        var i = Number(el.dataset.seg);
        if (!segs[i]) return;
        if (el.dataset.sf === 'color') segs[i].color = el.value;
        else segs[i][el.dataset.sf] = el.value === '' ? null : Number(el.value);
        emit({ segments: segs });
      };
    });
    host.querySelectorAll('[data-seg-del]').forEach(function (btn) {
      btn.onclick = function () {
        var segs = (spec.segments || []).slice();
        segs.splice(Number(btn.dataset.segDel), 1);
        emit({ segments: segs });
      };
    });
    var addSeg = host.querySelector('[data-k="add_segment"]');
    if (addSeg) addSeg.onclick = function () {
      emit({ segments: (spec.segments || []).concat([{ min: null, max: null, color: COLORS[(spec.segments || []).length % COLORS.length] }]) });
    };
  }

  function scalarColor(spec, value) {
    var color = spec.color || '#111';
    var n = Number(value);
    (spec.segments || []).forEach(function (s) {
      if (!isNum(value)) return;
      if (s.min != null && s.min !== '' && n < Number(s.min)) return;
      if (s.max != null && s.max !== '' && n > Number(s.max)) return;
      if (s.color) color = s.color;
    });
    return color;
  }
  function renderScalar(host, columns, rows, spec) {
    var name = (spec.y && spec.y[0]) || columns[0];
    var i = colIndex(columns, name);
    var value = rows && rows[0] ? rows[0][i] : null;
    var goal = spec.goal;
    var extra = '';
    if (goalOn(spec) && isNum(value) && isNum(goal) && Number(goal) !== 0) {
      extra = '<em>' + esc(formatNumber((Number(value) / Number(goal)) * 100, { decimals: 1, suffix: '% 到达目标' })) + '</em>';
    }
    host.innerHTML = '<div class="viz-scalar"><b style="color:' + esc(scalarColor(spec, value)) + '">' + esc(formatNumber(value, spec)) + '</b><small>' +
      esc(titleOf(spec, name) || name || '结果') +
      (goalOn(spec) ? ' · ' + esc(spec.goal_label || '目标') + ' ' + esc(formatNumber(goal, spec)) : '') + '</small>' + extra + '</div>';
  }
  function render(host, opts) {
    opts = opts || {};
    if (typeof host === 'string') host = document.querySelector(host);
    if (!host) return;
    disposeHost(host);
    var columns = opts.columns || [];
    var rows = opts.rows || [];
    var spec = infer(columns, rows, opts.queryir, opts.spec);
    var view = project(columns, rows, spec);
    if (opts.onSpec) opts.onSpec(spec);
    if (!columns.length) {
      host.innerHTML = '<div class="viz-empty"><b>还没有结果</b><p>这条查询没有返回字段。</p></div>';
      return spec;
    }
    if (spec.type === 'table') {
      var state = tableViewState(spec);
      host.innerHTML = '<div class="tb-grid-host"></div>';
      if (!global.TopbaseGrid) return spec;
      if (opts.compact) {
        global.TopbaseGrid(host.firstChild, { columns: view.columns, rows: view.rows, aliases: Object.assign({}, opts.aliases || {}, view.aliases), compact: true, dashboardOnly: !!opts.dashboardOnly, filtersEnabled: false });
      } else {
        global.TopbaseGrid(host.firstChild, {
          columns: columns,
          rows: rows,
          aliases: Object.assign({}, opts.aliases || {}, state.aliases),
          hidden: state.hidden,
          filters: state.filters,
          filtersEnabled: opts.tableFilters !== false,
          search: state.search,
          sort: state.sort,
          dir: state.dir,
          onChange: opts.onViewChange,
          compact: false
        });
      }
      return spec;
    }
    columns = view.columns;
    rows = view.rows;
    spec = infer(columns, rows, opts.queryir, spec);
    if (spec.type === 'scalar') {
      renderScalar(host, columns, rows, spec);
      return spec;
    }
    if (!rows.length) {
      host.innerHTML = '<div class="viz-empty"><b>没有可展示的行</b><p>查询已完成，但结果是空的。可以放宽筛选后再试。</p></div>';
      return spec;
    }
    var packed = seriesValues(columns, rows, spec);
    if (spec.type === 'pie') packed = capSlices(packed, spec);
    else if (spec.max_categories) packed = capCategories(packed, spec);
    mountChart(host, chartOption(spec, packed, !!opts.compact), spec);
    return spec;
  }

  global.TopbaseViz = {
    types: TYPES,
    infer: infer,
    merge: merge,
    matchFilter: matchFilter,
    project: project,
    render: render,
    renderTypes: renderTypes,
    renderSettings: renderSettings
  };
})(window);
