let databases=[], tables=[], active=null, currentDB=null, lastQueryIR=null, lastChart=null, lastResultMode='visual', isAdmin=false, aliases={}, fieldMeta=[], filterBar=null, gridState={hidden:{}}, creationMode=false, queryEditor=null, editingQuestion=null;
let navigationRequest = null;
let resultRequest = null;

function beginNavigation() {
  navigationRequest?.abort();
  resultRequest?.abort();
  $('#save-question').disabled = false;
  navigationRequest = new AbortController();
  return navigationRequest.signal;
}

// Every producer of the result grid shares one slot. Even when a transport
// finishes after cancellation, only the latest result can update the editor.
async function loadResult(request, applyResult, onError) {
  resultRequest?.abort();
  const controller = new AbortController();
  resultRequest = controller;
  $('#save-question').disabled = true;
  try {
    const result = await request(controller.signal);
    if (!controller.signal.aborted) applyResult(result);
  } catch (error) {
    if (!controller.signal.aborted) onError(error);
  } finally {
    if (resultRequest === controller) {
      resultRequest = null;
      $('#save-question').disabled = false;
    }
  }
}
function key(t){return t.schema+'.'+t.name}
function setPageContext(fallback){
  const label=editingQuestion?'编辑分析':(creationMode?'新建分析':fallback);
  document.title='Topbase · '+label;
}
function setURL(){
  const p=new URLSearchParams();
  if(editingQuestion)p.set('edit',editingQuestion.id);
  else if(creationMode)p.set('from','new-analysis');
  if(currentDB) p.set('db', currentDB.id);
  if(active){p.set('schema', active.schema);p.set('table', active.name)}
  history.replaceState({}, '', '/data/'+(p.toString()?'?'+p:''));
}
function baseField(value){return String(value||'').split('.').pop()}
async function hydrateQuestion(question){
  if(question.query_type==='native'){
    queryEditor.setMode('sql');
    queryEditor.setSQL(question.native_sql||'',{dirty:false});
    return;
  }
  const query=question.queryir;
  if(!query)return;
  lastQueryIR=query;lastChart=question.chartspec||lastChart;
  const join=(query.joins||[])[0];
  if(join&&join.table){
    const target=tables.find(table=>table.schema===join.table.schema&&table.name===join.table.name);
    if(target){
      $('#join-table').value=key(target);updateJoinTarget();
      $('#join-type').value=join.type||'left';
      const condition=(join.conditions||[])[0]||{};
      $('#join-left-field').value=baseField(condition.left);$('#join-right-field').value=baseField(condition.right);
      const joinedFields=(query.fields||[]).filter(field=>String(field).startsWith(target.name+'.')).map(baseField);
      $$('#join-fields input').forEach(input=>input.checked=joinedFields.includes(input.value));
    }
  }
  const selected=(query.fields||[]).map(baseField);
  if(selected.length)$$('#fields input').forEach(input=>input.checked=selected.includes(input.value));
  const aggregation=(query.aggregations||[])[0];
  if(aggregation){
    $('#aggregation').value=aggregation.fn||'';updateAggregationControls();
    $('#aggregation-field').value=baseField(aggregation.field);
    const group=(query.group_by||[])[0];
    if(group){$('#group-by-field').value=baseField(group.field);updateGroupControls();$('#group-by-temporal').value=group.temporal||''}
  }
  const expression=(query.expressions||[])[0];
  if(expression){$('#expression-alias').value=expression.alias||'';$('#expression-left').value=baseField(expression.left);$('#expression-op').value=expression.op||'add';$('#expression-right').value=String(expression.right??'')}
  $('#row-limit').value=String(query.limit||1000);
  const order=(query.order_by||[])[0];
  if(order){
    const group=(query.group_by||[])[0];
    const aggregation=(query.aggregations||[])[0];
    const value=order.field===(aggregation&&aggregation.fn)?'__metric__':(group&&order.field===baseField(group.field)?'__group__':baseField(order.field));
    refreshSortOptions();if($$('#sort-field option').some(option=>option.value===value))$('#sort-field').value=value;
    $('#sort-direction').value=order.dir==='desc'?'desc':'asc';
  }
  mountFilter();
  updateSelectedFieldCount();updateBuilderSummary();
  await loadResult(signal=>api('/api/dataset','POST',query,{signal}), result=>{
    lastQueryIR=result.queryir||query;
    lastChart=question.chartspec||result.chartspec;
    renderGrid(result,'visual');
  }, error=>{
    $('#grid-status').textContent=TopbaseLocale.t('无法运行已保存的查询：{error}',{error:error.message});
    $('#grid-wrap').innerHTML='';
    toast(error.message);
  });
}
function showDatabases(){
  beginNavigation();
  currentDB=null;active=null;tables=[];
  $('#db-view').hidden=false;$('#table-view').hidden=true;
  setPageContext('源数据');
  setURL();
  renderDatabases();
}
function renderDatabases(){
  const q=($('#db-search').value||'').toLowerCase();
  const items=databases.filter(d=>(d.name+' '+(d.host||'')).toLowerCase().includes(q));
  $('#db-cards').innerHTML=items.map(d=>`<button class="db-card" data-id="${esc(d.id)}" type="button"><b translate="no">${esc(d.name)}</b><small><span translate="no">${esc(d.engine||'PostgreSQL')} · ${esc(d.host||'')}</span> · ${esc(TopbaseLocale.t('表数：{n}',{n:d.table_count||0}))}<br>已连接</small></button>`).join('')||emptyHTML({icon:'▣',title:'还没有数据源',body:'管理员可在管理后台连接数据库。',href:'/admin/',cta:'打开数据源管理'});
  $$('#db-cards [data-id]').forEach(card=>card.onclick=()=>openDatabase(card.dataset.id).catch(e=>toast(e.message)));
}
async function openDatabase(id){
  const signal = beginNavigation();
  currentDB=databases.find(d=>d.id===id)||{id};
  topbaseRememberDatabase(id);
  active=null;tables=[];
  $('#search').value='';
  $('#table-count').textContent='';
  $('#schema-status').textContent='';
  $('#schema-status').hidden=true;
  $('#table-list').textContent='正在读取表…';
  $('#db-view').hidden=true;$('#table-view').hidden=false;
  $('#db-title').translate=false;
  $('#db-title').textContent=currentDB.name||id;
  setPageContext(currentDB.name||id);
  $('#empty-state').hidden=false;$('#table-workspace').hidden=true;
  setURL();
  let items;
  try{
    items=await api('/api/databases/'+id+'/tables','GET',null,{signal});
  }catch(e){
    if(signal.aborted)return false;
    try{items=await api('/api/databases/'+id+'/tables?cached=1','GET',null,{signal})}catch(_){items=[]}
    if(signal.aborted)return false;
    if(!items.length){
      $('#table-list').innerHTML=`<div class="reconnect"><b>无法读取表结构</b><small>${esc(e.message)}</small>${isAdmin?`<a href="/admin/?id=${encodeURIComponent(id)}">前往数据源管理</a>`:''}</div>`;
      return false;
    }
    $('#schema-status').textContent='当前显示缓存的表结构';
    $('#schema-status').hidden=false;
  }
  if(signal.aborted)return false;
  tables=items;
  $('#table-count').textContent=tables.length;
  renderTables(tables.filter(t=>!t.hidden));
  return true;
}
function renderTables(items){
  const groups={};
  items.forEach(t=>{ (groups[t.schema]||(groups[t.schema]=[])).push(t) });
  const html=Object.keys(groups).sort().map(schema=>{
    const rows = groups[schema].map(table => {
      const selected = active && key(active) === key(table);
      const title = table.display_name || table.name;
      const badge = table.warehouse ? ' · ' + TopbaseLocale.t('已沉淀') : '';
      const detail = table.display_name ? table.schema + '.' + table.name
        : table.description || TopbaseLocale.t('字段：{n}', {n: (table.columns || []).length});
      return `<button translate="no" class="table-item ${selected ? 'active' : ''}" data-key="${esc(key(table))}" ${selected ? 'aria-current="true"' : ''} title="${esc(table.description || '')}">
        <b>${esc(title + badge)}</b><small>${esc(detail)}</small></button>`;
    }).join('');
    return `<div class="schema-label" translate="no">${esc(schema)}</div>${rows}`;
  }).join('');
  $('#table-list').innerHTML=html||'没有可读取的数据表。';
  $$('#table-list .table-item').forEach(b=>b.onclick=()=>openTable(tables.find(t=>key(t)===b.dataset.key)).catch(e=>toast(e.message)));
  const selected=$('#table-list .table-item.active');
  if(selected)requestAnimationFrame(()=>selected.scrollIntoView({block:'center',inline:'nearest'}));
}
async function openTable(table, question){
  const signal = beginNavigation();
  active=table;lastQueryIR=null;lastChart=null;lastResultMode='visual';aliases={};fieldMeta=[];gridState={hidden:{}};
  if(filterBar){filterBar.destroy();filterBar=null}
  queryEditor.setMode('visual');queryEditor.setSQL('',{dirty:false});queryEditor.setGeneratedSQL('');
  renderTables(tables.filter(t=>!t.hidden));
  $('#empty-state').hidden=true;$('#table-workspace').hidden=false;$('#ask-panel').hidden=false;
  $('#toggle-ask').textContent='收起查询编辑器';
  $('#toggle-ask').setAttribute('aria-expanded','true');
  $('#source-name').textContent=table.schema;
  $('#table-name').textContent=table.display_name||table.name;
  $('#table-description').textContent=TopbaseLocale.t('正在拉取数据…');
  $('#grid-status').textContent='正在拉取数据…';
  $('#grid-wrap').innerHTML='';
  setPageContext(table.display_name||table.name);
  $('#edit-meta').hidden=!isAdmin;
  $('#edit-meta').href=`/admin/datamodel/?db=${encodeURIComponent(currentDB.id)}&schema=${encodeURIComponent(table.schema)}&table=${encodeURIComponent(table.name)}`;
  setURL();
  const cols=table.columns||[];
  $('#builder-source').textContent=table.schema+'.'+table.name;
  $('#fields').innerHTML=cols.map(c=>`<label translate="no" title="${esc(c.description||'')}"><input type="checkbox" value="${esc(c.name)}" checked> <span>${esc(c.name)}</span></label>`).join('');
  const options='<option value="">选择字段</option>'+cols.map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#aggregation').value='';
  $('#aggregation-field').innerHTML='<option value="">记录数无需选择字段</option>'+cols.map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#aggregation-field').hidden=true;
  $('#group-by-field').innerHTML='<option value="">不分组</option>'+cols.map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#group-by-field').value='';
  $('#group-by-temporal').value='';$('#group-by-temporal').hidden=true;
  $('#sort-field').innerHTML='<option value="">不排序</option>'+cols.map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#sort-field').value='';$('#sort-direction').value='asc';
  $('#row-limit').value='1000';
  $('#expression-alias').value='';$('#expression-right').value='';$('#expression-op').value='add';
  $('#expression-left').innerHTML=options;
  $('#expression-field-options').innerHTML=cols.map(c=>`<option translate="no" value="${esc(c.name)}"></option>`).join('');
  $('#drill-field').innerHTML='<option value="">当前筛选字段</option>'+cols.map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $$('.step-editor').forEach(panel=>panel.hidden=true);
  $$('[data-builder-target]').forEach(button=>button.classList.remove('active'));
  updateSelectedFieldCount();
  updateBuilderSummary();
  const id=currentDB.id;
  const query={version:1,source:{database_id:id,table:{schema:table.schema,name:table.name}},limit:1000};
  const metadata=Promise.all([
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/annotation`,'GET',null,{signal}).catch(()=>({})),
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/fields`,'GET',null,{signal}).catch(()=>[])
  ]);
  const preview=question ? null : loadResult(async resultSignal=>{
    const [result]=await Promise.all([api('/api/dataset','POST',query,{signal:resultSignal}),metadata]);
    return result;
  },result=>{
    lastQueryIR=result.queryir||query;
    lastChart=result.chartspec;
    renderGrid(result,'visual');
  },error=>{
    $('#grid-status').textContent=TopbaseLocale.t('读取失败：{error}',{error:error.message});
    $('#grid-wrap').innerHTML=`<div class="reconnect"><b>查询失败</b><small>${esc(error.message)}</small></div>`;
  });
  const [note, fields]=await metadata;
  if(signal.aborted)return false;
  fieldMeta=fields||[];
  fieldMeta.forEach(f=>{if(f.display_name)aliases[f.name]=f.display_name});
  $('#table-name').textContent=note.display_name||table.name;
  $('#table-description').textContent=note.user_note||table.description||(table.schema+'.'+table.name);
  lastQueryIR=query;
  configureJoinBuilder();
  mountFilter();
  if(question){
    $('#grid-status').textContent=question.query_type==='native'?'运行 SQL 即可预览结果。':'正在运行已保存的分析…';
    await hydrateQuestion(question);
  }else{
    await preview;
  }
  return !signal.aborted;
}
function columnModels(){
  const metadata = new Map(fieldMeta.map(field=>[field.name,field]));
  return (active&&active.columns||[]).filter(column=>metadata.get(column.name)?.visibility!=='hidden').map(c=>{
    const meta=metadata.get(c.name)||{};
    return {
      name:c.name,
      data_type:c.data_type,
      semantic_type:meta.semantic_type||(c.primary_key?'EntityKey':(c.foreign_key?'ForeignKey':'')),
      display_name:aliases[c.name]||meta.display_name||c.name,
      description:meta.description||c.description||''
    };
  });
}
function tableForJoin(value){return tables.find(t=>key(t)===value)}
function optionList(columns, selected, placeholder){return `<option value="">${placeholder||'选择字段'}</option>`+(columns||[]).map(c=>`<option translate="no" value="${esc(c.name)}" ${c.name===selected?'selected':''}>${esc(c.name)}</option>`).join('')}
function isDateField(name){
  const column=(active&&active.columns||[]).find(c=>c.name===name);
  return !!(column&&/date|time|timestamp/i.test(column.data_type||''));
}
function qualifyField(name,joined){return name&&joined&&!name.includes('.')?active.name+'.'+name:name}
function updateSelectedFieldCount(){
  if(!active)return;
  const checked=$$('#fields input:checked').length,total=(active.columns||[]).length;
  $('#selected-field-count').textContent=checked===total?'全部字段':TopbaseLocale.t('已选字段：{selected} / {total}',{selected:checked,total});
}
function updateAggregationControls(){
  const aggregation=$('#aggregation').value;
  $('#aggregation-field').hidden=!aggregation||aggregation==='count';
  refreshSortOptions();
  updateBuilderSummary();
}
function updateGroupControls(){
  const field=$('#group-by-field').value;
  $('#group-by-temporal').hidden=!field||!isDateField(field);
  if($('#group-by-temporal').hidden)$('#group-by-temporal').value='';
  refreshSortOptions();
  updateBuilderSummary();
}
function breakoutAlias(field,temporal,joined){
  const qualified=qualifyField(field,joined)||'';
  const base=qualified.replaceAll('.','_');
  return temporal?base+'_'+temporal:qualified;
}
function refreshSortOptions(){
  if(!active)return;
  const selected=$('#sort-field').value;
  const aggregation=$('#aggregation').value;
  const group=$('#group-by-field').value;
  let options='<option value="">不排序</option>';
  if(aggregation){
    options+=`<option value="__metric__">${esc(TopbaseLocale.t('汇总结果（{name}）',{name:aggregation}))}</option>`;
    if(group)options+=`<option value="__group__">${esc(TopbaseLocale.t('分组字段（{name}）',{name:group}))}</option>`;
  }else{
    options+=(active.columns||[]).map(c=>`<option translate="no" value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  }
  $('#sort-field').innerHTML=options;
  if($$('#sort-field option').some(option=>option.value===selected))$('#sort-field').value=selected;
}
function expressionDraft(joined){
  const alias=$('#expression-alias').value.trim();
  const left=$('#expression-left').value;
  const raw=$('#expression-right').value.trim();
  if(!alias&&!left&&!raw)return null;
  if(!/^[A-Za-z_][A-Za-z0-9_$]*$/.test(alias))throw Error('自定义字段名仅支持字母、数字和下划线，且不能以数字开头。');
  if(!left||!raw)throw Error('请完整填写自定义字段的左侧字段和右侧内容。');
  const baseColumn=(active.columns||[]).some(column=>column.name===raw);
  const number=Number(raw);
  const right=baseColumn?qualifyField(raw,joined):(raw!==''&&Number.isFinite(number)?number:raw);
  return {alias,op:$('#expression-op').value,left:qualifyField(left,joined),right};
}
function updateBuilderSummary(){
  if(!active)return;
  const filters=filterBar?filterBar.filters().length:0;
  const aggregation=$('#aggregation').value;
  const group=$('#group-by-field').value;
  const joined=!!$('#join-table').value;
  const sort=!!$('#sort-field').value;
  const parts=[];
  if(filters)parts.push(TopbaseLocale.t('筛选：{n}',{n:filters}));
  if(aggregation)parts.push(TopbaseLocale.t('{group} · {operation}',{group:group||TopbaseLocale.t('全部记录'),operation:aggregation==='count'?TopbaseLocale.t('计数'):aggregation}));
  if(joined)parts.push(TopbaseLocale.t('跨表关联'));
  if(sort)parts.push(TopbaseLocale.t('结果排序'));
  if($('#expression-alias').value.trim())parts.push(TopbaseLocale.t('自定义字段'));
  if(Number($('#row-limit').value)!==1000)parts.push(TopbaseLocale.t('最多 {n} 行',{n:Number($('#row-limit').value)||1000}));
  const text=parts.length?parts.join(' · '):TopbaseLocale.t('显示原始数据');
  $('#query-summary').textContent=text;
}
function configureJoinBuilder(){
  if(!active)return;
  const picker=$('#join-table');
  const candidates=tables.filter(t=>key(t)!==key(active));
  picker.innerHTML='<option value="">不关联</option>'+candidates.map(t=>`<option translate="no" value="${esc(key(t))}">${esc(t.schema+'.'+t.name)}</option>`).join('');
  $('#join-left-field').innerHTML=optionList(active.columns);
  $('#join-right-field').innerHTML='<option value="">请先选择关联表</option>';
  $('#join-fields').textContent='请先选择关联表。';
  $('#join-hint').textContent='选择关联表，再选择两表的匹配字段。';
}
function updateJoinTarget(){
  const target=tableForJoin($('#join-table').value);
  const joinButton=$('[data-builder-target="join-builder"]');
  if(joinButton)joinButton.classList.toggle('active',!!target);
  if(!target){
    $('#join-right-field').innerHTML='<option value="">请先选择关联表</option>';
    $('#join-fields').textContent='请先选择关联表。';
    updateBuilderSummary();
    return;
  }
  const manualRelation=fieldMeta.find(f=>f.fk_target&&f.fk_target.table===target.name&&(f.fk_target.schema===''||f.fk_target.schema===target.schema));
  const sourceColumn=(active.columns||[]).find(c=>c.foreign_key&&c.foreign_key.table===target.name&&(!c.foreign_key.schema||c.foreign_key.schema===target.schema));
  const relation=manualRelation||(sourceColumn?{name:sourceColumn.name,fk_target:sourceColumn.foreign_key}:null);
  $('#join-left-field').innerHTML=optionList(active.columns,relation&&relation.name);
  $('#join-right-field').innerHTML=optionList(target.columns,relation&&relation.fk_target&&relation.fk_target.name);
  $('#join-fields').innerHTML=target.columns.map(c=>`<label translate="no"><input type="checkbox" value="${esc(c.name)}"> ${esc(c.name)} <small>${esc(c.data_type||'')}</small></label>`).join('');
  $('#join-hint').textContent=relation?'已按外键选择匹配字段，可手动调整。':'请选择两表的匹配字段。';
  updateBuilderSummary();
}
function selectedJoin(){
  const target=tableForJoin($('#join-table').value);
  const left=$('#join-left-field').value,right=$('#join-right-field').value;
  if(!target)return {join:null,fields:[]};
  if(!left||!right)throw Error('请为关联关系分别选择当前表字段和关联表字段。');
  return {join:{type:$('#join-type').value,alias:target.name,table:{schema:target.schema,name:target.name},conditions:[{left:active.name+'.'+left,right:target.name+'.'+right,op:'='}]},fields:$$('#join-fields input:checked').map(box=>target.name+'.'+box.value)};
}
function qualifyBaseFilters(filters, joined){
  if(!joined)return filters||[];
  return (filters||[]).map(f=>Object.assign({},f,{field:f.field&&f.field.includes('.')?f.field:active.name+'.'+f.field}));
}
async function fetchDistinct(field){
  if(!lastQueryIR||!lastQueryIR.source)return [];
  const signal=navigationRequest.signal;
  const others=(lastQueryIR.filters||[]).filter(f=>f.field!==field&&f.field!==active.name+'.'+field);
  const sourceField=lastQueryIR.joins&&lastQueryIR.joins.length?active.name+'.'+field:field;
  const d=await api('/api/dataset','POST',{
    version:1,
    source:lastQueryIR.source,
    aggregations:[{fn:'count'}],
    joins:lastQueryIR.joins||[],
    group_by:[{field:sourceField}],
    order_by:[{field:'count',dir:'desc'}],
    filters:others,
    limit:80
  },{signal});
  signal.throwIfAborted();
  return (d.rows||[]).map(r=>r[0]).filter(v=>v!==null&&v!==undefined);
}
function mountFilter(){
  if(filterBar) filterBar.destroy();
  filterBar=TopbaseFilter('#filter-bar',{
    columns:columnModels(),
    filters:lastQueryIR&&lastQueryIR.filters||[],
    fetchValues:fetchDistinct,
    onChange:filters=>{updateBuilderSummary();runWithFilters(filters)}
  });
}
async function runWithFilters(filters){
  if(!lastQueryIR)return;
  const query=Object.assign({}, lastQueryIR, {filters:qualifyBaseFilters(filters,(lastQueryIR.joins||[]).length>0), limit:lastQueryIR.limit||Number($('#row-limit').value)||1000});
  $('#grid-status').textContent='正在按筛选条件查询…';
  await loadResult(signal=>api('/api/dataset','POST',query,{signal}), d=>{
    lastQueryIR=d.queryir||query;
    lastChart=d.chartspec;
    renderGrid(d,'visual');
  },e=>{
    $('#grid-status').textContent=TopbaseLocale.t('筛选失败：{error}',{error:e.message});
    toast(e.message);
  });
}
function renderGrid(d, mode){
  mode=mode||lastResultMode||'visual';
  lastResultMode=mode;
  const cols=d.columns||[];
  const rows=d.rows||[];
  queryEditor.setGeneratedSQL(d.sql||'');
  const n=mode==='visual'?(lastQueryIR&&lastQueryIR.filters||[]).length:0;
  const limited=queryLimitMessage(d.meta);
  $('#grid-status').textContent=TopbaseLocale.t('数据库返回 {n} 行',{n:rows.length})+(mode==='sql'?' · SQL':(n?' · '+TopbaseLocale.t('筛选：{n}',{n}):''))+(limited?` · ${limited}`:'');
  if(!cols.length){$('#grid-wrap').innerHTML='<p>这张表没有返回列。</p>';return}
  const types={},semanticTypes=Object.assign({},d.meta&&d.meta.semantic_types||{}),descriptions={};
  tables.forEach(table=>(table.columns||[]).forEach(column=>{
    if(types[column.name]===undefined)types[column.name]=column.data_type;
    if(descriptions[column.name]===undefined)descriptions[column.name]=column.description||'';
  }));
  fieldMeta.forEach(field=>{if(field.description)descriptions[field.name]=field.description;if(field.semantic_type)semanticTypes[field.name]=field.semantic_type});
  TopbaseGrid('#grid-wrap',{columns:cols, rows, aliases, types, semanticTypes, descriptions, filtersEnabled:false, hideToolbar:true, hidden:gridState.hidden, onChange:state=>{gridState.hidden=state.hidden}});
}
queryEditor=TopbaseQueryEditor.mount('#ask-panel',{
  onSQLChange:()=>{lastChart=null},
  onModeChange:mode=>{
    $$('.visual-result-action').forEach(item=>item.hidden=mode==='sql');
    if(mode==='sql'){
      queryEditor&&queryEditor.setSummary('执行自定义 SQL');
    }else{
      updateBuilderSummary();
    }
  },
  onRun:mode=>mode==='sql'?runNativeSQL():runVisualQuery()
});
$('#back-dbs').onclick=()=>editingQuestion?location.assign('/questions/'+editingQuestion.id+'/'):(creationMode?location.assign('/questions/new/'):showDatabases());
$('#db-search').oninput=renderDatabases;
$('#search').oninput=e=>renderTables(tables.filter(t=>!t.hidden&&((t.display_name||'')+' '+t.name+' '+t.schema+' '+(t.description||'')+' '+(t.columns||[]).map(c=>c.name+' '+(c.description||'')).join(' ')).toLowerCase().includes(e.target.value.toLowerCase())));
$('#toggle-ask').onclick=()=>{
  const panel=$('#ask-panel');
  panel.hidden=!panel.hidden;
  $('#toggle-ask').textContent=panel.hidden?'展开查询编辑器':'收起查询编辑器';
  $('#toggle-ask').setAttribute('aria-expanded',panel.hidden?'false':'true');
};
$('#join-table').onchange=updateJoinTarget;
$('#fields').onchange=()=>{updateSelectedFieldCount();updateBuilderSummary()};
document.addEventListener('click',event=>{
  const details=$('.field-details');
  if(details&&details.open&&!details.contains(event.target))details.open=false;
});
document.addEventListener('keydown',event=>{
  if(event.key!=='Escape')return;
  const details=$('.field-details');
  if(!details||!details.open)return;
  details.open=false;
  details.querySelector('summary')?.focus();
});
$('#aggregation').onchange=updateAggregationControls;
$('#aggregation-field').onchange=updateBuilderSummary;
$('#group-by-field').onchange=updateGroupControls;
$('#group-by-temporal').onchange=()=>{refreshSortOptions();updateBuilderSummary()};
$('#sort-field').onchange=updateBuilderSummary;
$('#sort-direction').onchange=updateBuilderSummary;
$('#row-limit').oninput=updateBuilderSummary;
$('#expression-alias').oninput=updateBuilderSummary;
$('#expression-left').onchange=updateBuilderSummary;
$('#expression-op').onchange=updateBuilderSummary;
$('#expression-right').oninput=updateBuilderSummary;
$$('[data-builder-target]').forEach(button=>button.onclick=()=>{
  const panel=$('#'+button.dataset.builderTarget);
  const opening=panel.hidden;
  $$('.step-editor').forEach(item=>{if(item!==panel)item.hidden=true});
  $$('[data-builder-target]').forEach(item=>{if(item!==button)item.classList.remove('active')});
  panel.hidden=!opening;
  button.classList.toggle('active',opening||button.dataset.builderTarget==='join-builder'&&!!$('#join-table').value);
  if(opening)panel.scrollIntoView({behavior:'smooth',block:'nearest'});
});
$$('[data-close-step]').forEach(button=>button.onclick=()=>{
  const panel=$('#'+button.dataset.closeStep);panel.hidden=true;
  const trigger=$(`[data-builder-target="${button.dataset.closeStep}"]`);
  if(trigger&&(button.dataset.closeStep!=='join-builder'||!$('#join-table').value))trigger.classList.remove('active');
});
async function runVisualQuery(){
  if(!active||!currentDB||!lastQueryIR)return;
  let join;
  try{join=selectedJoin()}catch(e){toast(e.message);return}
  const chosen=$$('#fields input:checked').map(x=>x.value);
  const joined=!!join.join;
  const selected=chosen.length?chosen:active.columns.map(c=>c.name);
  const fields=(joined?selected.map(name=>qualifyField(name,true)):selected).concat(join.fields);
  const aggregation=$('#aggregation').value;
  const aggregationField=$('#aggregation-field').value;
  const groupField=$('#group-by-field').value;
  const groupTemporal=$('#group-by-temporal').value;
  const limit=Math.max(1,Math.min(10000,Number($('#row-limit').value)||1000));
  let expression;
  try{expression=expressionDraft(joined)}catch(e){toast(e.message);return}
  const query={
    version:1,
    source:Object.assign({},lastQueryIR.source,joined?{alias:active.name}:{alias:''}),
    fields:aggregation?[]:fields,
    joins:joined?[join.join]:[],
    filters:qualifyBaseFilters(filterBar?filterBar.filters():lastQueryIR.filters||[],joined),
    limit
  };
  if(aggregation){
    if(aggregation!=='count'&&!aggregationField){toast('请选择需要汇总的字段。');return}
    query.aggregations=[{fn:aggregation,field:aggregationField?qualifyField(aggregationField,joined):''}];
    if(groupField)query.group_by=[{field:qualifyField(groupField,joined),temporal:groupTemporal||''}];
  }else if(expression){
    query.expressions=[expression];
  }
  const sortField=$('#sort-field').value;
  if(sortField){
    let field=qualifyField(sortField,joined);
    if(sortField==='__metric__')field=aggregation;
    if(sortField==='__group__')field=breakoutAlias(groupField,groupTemporal,joined);
    query.order_by=[{field,dir:$('#sort-direction').value}];
  }
  $('#grid-status').textContent='正在运行查询…';
  await loadResult(signal=>api('/api/dataset','POST',query,{signal}), d=>{
    lastQueryIR=d.queryir||query;
    lastChart=d.chartspec;
    renderGrid(d,'visual');updateBuilderSummary();toast('查询已完成');
  },e=>{$('#grid-status').textContent=TopbaseLocale.t('查询失败：{error}',{error:e.message});toast(e.message)});
}
async function runNativeSQL(){
  if(!active||!currentDB)return;
  const sql=queryEditor.sql().trim();
  if(!sql){toast('请先输入要执行的 SQL');return}
  $('#grid-status').textContent='正在执行 SQL…';
  await loadResult(signal=>api('/api/queries/run','POST',{database_id:currentDB.id,sql},{signal}), d=>{
    lastChart=null;
    renderGrid(Object.assign({},d,{sql:d.sql||sql}),'sql');
    toast('SQL 查询已完成');
  },e=>{$('#grid-status').textContent=TopbaseLocale.t('SQL 查询失败：{error}',{error:e.message});toast(e.message)});
}
$('#save-question').onclick=async()=>{
  if(!lastQueryIR&&!queryEditor.sql().trim())return toast('请先打开一张表并完成查询');
  const signal=navigationRequest.signal;
  let collections;
  try{collections=await api('/api/collections')}catch(e){if(!signal.aborted)toast(TopbaseLocale.t('读取分组失败：{error}',{error:e.message}));return}
  if(signal.aborted)return;
  const writableCollections=collections.filter(collection=>!collection.read_only);
  const personal=writableCollections.find(collection=>collection.kind==='personal_project');
  const values=await formDialog({
    kicker:'保存查询',
    title:editingQuestion?'保存分析修改':'将当前查询保存为分析',
    description:'为分析命名，并选择它在分析列表中的分组。',
    confirmText:editingQuestion?'保存修改':'保存分析',
    fields:[
      {name:'name',label:'分析名称',value:(editingQuestion&&editingQuestion.name)||(active&&($('#table-name').textContent||active.name))||TopbaseLocale.t('未命名分析'),placeholder:'例如：活跃客户明细',required:true},
      {name:'collection_id',label:'保存到分组',type:'select',value:(editingQuestion&&editingQuestion.collection_id)||(personal?personal.id:'__personal__'),required:true,options:(personal?[]:[{value:'__personal__',label:'我的分析（默认分组）'}]).concat(writableCollections.map(collection=>({value:collection.id,translate:false,label:collection.name+' · '+TopbaseLocale.t(collection.kind==='personal_project'?'个人分组':'企业项目')})))}
    ]
  });
  if(!values||signal.aborted)return;
  const sql=queryEditor.sql().trim();
  const payload=queryEditor.mode()==='sql'
    ?{name:values.name,collection_id:values.collection_id==='__personal__'?'':values.collection_id,query_type:'native',database_id:currentDB.id,native_sql:sql,chartspec:lastChart}
    :{name:values.name,collection_id:values.collection_id==='__personal__'?'':values.collection_id,query_type:'queryir',queryir:lastQueryIR,chartspec:lastChart};
  if(payload.query_type==='native'&&!sql)return toast('请先输入要保存的 SQL');
  try{const saved=editingQuestion?await api('/api/questions/'+editingQuestion.id,'PUT',payload):await api('/api/questions','POST',payload);toast(editingQuestion?'分析修改已保存':'已保存为分析');location.href='/questions/'+saved.id+'/'}catch(e){toast(e.message)}
};
async function drill(kind){
  if(queryEditor.mode()==='sql')return toast('SQL 结果暂不支持可视化下钻，请切换到可视化查询。');
  if(!lastQueryIR)return toast('请先查看表数据');
  const field=$('#drill-field').value|| (lastQueryIR.filters&&lastQueryIR.filters[0]&&lastQueryIR.filters[0].field) || (lastQueryIR.group_by&&lastQueryIR.group_by[0]&&lastQueryIR.group_by[0].field);
  const value=(lastQueryIR.filters&&lastQueryIR.filters[0]&&lastQueryIR.filters[0].value)||'';
  const selected=tableForJoin($('#join-table').value);
  await loadResult(signal=>api('/api/dataset/drill','POST',{queryir:lastQueryIR,drill:{kind,field,value},join_table:selected?selected.name:''},{signal}), d=>{
    if(d.queryir) lastQueryIR=d.queryir;
    if(filterBar) filterBar.setFilters(lastQueryIR.filters||[]);
    renderGrid(d,'visual');toast('下钻完成');
  },e=>toast(e.message));
}
$('#drill-records').onclick=()=>drill('records');
$('#drill-filter').onclick=()=>drill('filter');
async function boot(){
  try{
    const me=await api('/api/user/current').catch(()=>null);
    isAdmin=!!(me&&me.is_admin);
    $('#warehouse-link').hidden=!isAdmin;
    databases=await api('/api/databases');
    const params=new URLSearchParams(location.search);
    const editID=params.get('edit');
    if(editID){editingQuestion=await api('/api/questions/'+encodeURIComponent(editID));$('#back-dbs').textContent='← 返回分析';$('#toggle-ask').textContent='查询步骤';}
    creationMode=!!editingQuestion||params.get('from')==='new-analysis';
    if(creationMode&&!editingQuestion){$('#back-dbs').textContent='← 更换起始数据';$('#toggle-ask').textContent='查询步骤'}
    const source=editingQuestion&&editingQuestion.queryir&&editingQuestion.queryir.source;
    const db=params.get('db') || (source&&source.database_id) || (editingQuestion&&editingQuestion.database_id) || topbasePickDatabase(databases);
    renderDatabases();
    if(db && databases.some(d=>d.id===db)){
      topbaseRememberDatabase(db);
      if(!await openDatabase(db))return;
      const schema=params.get('schema') || (source&&source.table&&source.table.schema), table=params.get('table') || (source&&source.table&&source.table.name);
      if((schema&&table)||(editingQuestion&&editingQuestion.query_type==='native'&&tables.length)){
        const found=(schema&&table?tables.find(t=>t.schema===schema&&t.name===table):tables[0]);
        if(found)await openTable(found,editingQuestion);
      }
    }
  }catch(e){toast(e.message)}
}
boot();
