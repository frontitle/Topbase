let databases=[], tables=[], active=null, currentDB=null, lastQueryIR=null, lastChart=null, lastResultMode='visual', isAdmin=false, aliases={}, fieldMeta=[], filterBar=null, gridState={hidden:{}}, creationMode=false, queryEditor=null;
function key(t){return t.schema+'.'+t.name}
function setURL(){
  const p=new URLSearchParams();
  if(currentDB) p.set('db', currentDB.id);
  if(active){p.set('schema', active.schema);p.set('table', active.name)}
  history.replaceState({}, '', '/data/'+(p.toString()?'?'+p:''));
}
function showDatabases(){
  currentDB=null;active=null;tables=[];
  $('#db-view').hidden=false;$('#table-view').hidden=true;
  $('#crumb').textContent='选择数据库';
  setURL();
  renderDatabases();
}
function renderDatabases(){
  const q=($('#db-search').value||'').toLowerCase();
  const items=databases.filter(d=>(d.name+' '+(d.host||'')).toLowerCase().includes(q));
  $('#db-cards').innerHTML=items.map(d=>`<button class="db-card" data-id="${esc(d.id)}" type="button"><b>${esc(d.name)}</b><small>PostgreSQL · ${esc(d.host||'')} · ${d.table_count||0} 张表<br>已连接</small></button>`).join('')||emptyHTML({icon:'▣',title:'还没有数据源',body:'管理员可在管理后台连接数据库。',href:'/admin/',cta:'打开数据源管理'});
  $$('#db-cards [data-id]').forEach(card=>card.onclick=()=>openDatabase(card.dataset.id).catch(e=>toast(e.message)));
}
async function openDatabase(id){
  currentDB=databases.find(d=>d.id===id)||{id};
  topbaseRememberDatabase(id);
  active=null;
  $('#db-view').hidden=true;$('#table-view').hidden=false;
  $('#db-title').textContent=currentDB.name||id;
  $('#crumb').textContent=currentDB.name||id;
  $('#empty-state').hidden=false;$('#table-workspace').hidden=true;
  setURL();
  try{
    tables=await api('/api/databases/'+id+'/tables');
  }catch(e){
    try{tables=await api('/api/databases/'+id+'/tables?cached=1')}catch(_){tables=[]}
    $('#table-list').innerHTML=`<div class="reconnect"><b>无法读取表结构</b><small>${esc(e.message)}</small><a href="/admin/?id=${encodeURIComponent(id)}">前往数据源管理</a></div>`;
    toast(e.message);
    return;
  }
  $('#table-count').textContent=tables.length;
  renderTables(tables);
}
function renderTables(items){
  const groups={};
  items.forEach(t=>{ (groups[t.schema]||(groups[t.schema]=[])).push(t) });
  const html=Object.keys(groups).sort().map(schema=>{
    const rows=groups[schema].map(t=>`<button class="table-item ${active&&key(active)===key(t)?'active':''}" data-key="${esc(key(t))}" title="${esc(t.description||'')}"><b>${esc(t.name)}${t.warehouse?' · 数仓':''}</b><small>${esc(t.description||((t.columns||[]).length+' 个字段'))}</small></button>`).join('');
    return `<div class="schema-label">${esc(schema)}</div>${rows}`;
  }).join('');
  $('#table-list').innerHTML=html||'没有可读取的数据表。';
  $$('#table-list .table-item').forEach(b=>b.onclick=()=>openTable(tables.find(t=>key(t)===b.dataset.key)).catch(e=>toast(e.message)));
}
async function openTable(table){
  active=table;lastQueryIR=null;lastChart=null;lastResultMode='visual';aliases={};fieldMeta=[];gridState={hidden:{}};
  if(filterBar){filterBar.destroy();filterBar=null}
  queryEditor.setMode('visual');queryEditor.setSQL('',{dirty:false});queryEditor.setGeneratedSQL('');
  renderTables(tables);
  $('#empty-state').hidden=true;$('#table-workspace').hidden=false;$('#ask-panel').hidden=false;
  $('#toggle-ask').textContent='收起查询编辑器';
  $('#toggle-ask').setAttribute('aria-expanded','true');
  $('#source-name').textContent=table.schema;
  $('#table-name').textContent=table.name;
  $('#table-description').textContent='正在拉取数据…';
  $('#grid-status').textContent='正在拉取数据…';
  $('#grid-wrap').innerHTML='';
  $('#crumb').textContent=(currentDB.name||'')+' / '+table.schema+'.'+table.name;
  $('#edit-meta').hidden=!isAdmin;
  $('#edit-meta').href=`/admin/datamodel/?db=${encodeURIComponent(currentDB.id)}&schema=${encodeURIComponent(table.schema)}&table=${encodeURIComponent(table.name)}`;
  setURL();
  const cols=table.columns||[];
  $('#builder-source').textContent=table.schema+'.'+table.name;
  $('#fields').innerHTML=cols.map(c=>`<label title="${esc(c.description||'')}"><input type="checkbox" value="${esc(c.name)}" checked> <span>${esc(c.name)}</span></label>`).join('');
  const options='<option value="">选择字段</option>'+cols.map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#aggregation').value='';
  $('#aggregation-field').innerHTML='<option value="">记录数无需选择字段</option>'+cols.map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#aggregation-field').hidden=true;
  $('#group-by-field').innerHTML='<option value="">不分组</option>'+cols.map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#group-by-field').value='';
  $('#group-by-temporal').value='';$('#group-by-temporal').hidden=true;
  $('#sort-field').innerHTML='<option value="">不排序</option>'+cols.map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $('#sort-field').value='';$('#sort-direction').value='asc';
  $('#row-limit').value='1000';
  $('#expression-alias').value='';$('#expression-right').value='';$('#expression-op').value='add';
  $('#expression-left').innerHTML=options;
  $('#expression-field-options').innerHTML=cols.map(c=>`<option value="${esc(c.name)}"></option>`).join('');
  $('#drill-field').innerHTML='<option value="">当前筛选字段</option>'+cols.map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  $$('.step-editor').forEach(panel=>panel.hidden=true);
  $$('[data-builder-target]').forEach(button=>button.classList.remove('active'));
  updateSelectedFieldCount();
  updateBuilderSummary();
  const id=currentDB.id;
  const query={version:1,source:{database_id:id,table:{schema:table.schema,name:table.name}},limit:1000};
  const [note, fields, preview]=await Promise.all([
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/annotation`).catch(()=>({})),
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/fields`).catch(()=>[]),
    api('/api/dataset','POST',query).catch(err=>({error:err.message,columns:[],rows:[]}))
  ]);
  fieldMeta=fields||[];
  fieldMeta.forEach(f=>{if(f.display_name)aliases[f.name]=f.display_name});
  $('#table-name').textContent=note.display_name||table.name;
  $('#table-description').textContent=note.description||table.description||(table.schema+'.'+table.name);
  lastQueryIR=preview.queryir||query;
  lastChart=preview.chartspec;
  configureJoinBuilder();
  mountFilter();
  if(preview.error){
    $('#grid-status').textContent='无法拉取数据：'+preview.error;
    $('#grid-wrap').innerHTML=`<div class="reconnect"><b>查询失败</b><small>${esc(preview.error)}</small></div>`;
    return;
  }
  renderGrid(preview,'visual');
}
function columnModels(){
  return (active&&active.columns||[]).map(c=>{
    const meta=fieldMeta.find(f=>f.name===c.name)||{};
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
function optionList(columns, selected, placeholder){return `<option value="">${placeholder||'选择字段'}</option>`+(columns||[]).map(c=>`<option value="${esc(c.name)}" ${c.name===selected?'selected':''}>${esc(c.name)}</option>`).join('')}
function isDateField(name){
  const column=(active&&active.columns||[]).find(c=>c.name===name);
  return !!(column&&/date|time|timestamp/i.test(column.data_type||''));
}
function qualifyField(name,joined){return name&&joined&&!name.includes('.')?active.name+'.'+name:name}
function updateSelectedFieldCount(){
  if(!active)return;
  const checked=$$('#fields input:checked').length,total=(active.columns||[]).length;
  $('#selected-field-count').textContent=checked===total?'全部字段':`${checked} / ${total} 个字段`;
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
    options+=`<option value="__metric__">汇总结果（${esc(aggregation)}）</option>`;
    if(group)options+=`<option value="__group__">分组字段（${esc(group)}）</option>`;
  }else{
    options+=(active.columns||[]).map(c=>`<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
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
  if(filters)parts.push(`${filters} 条筛选`);
  if(aggregation)parts.push(`按${group||'全部记录'}${aggregation==='count'?'计数':aggregation}`);
  if(joined)parts.push('跨表关联');
  if(sort)parts.push('结果排序');
  if($('#expression-alias').value.trim())parts.push('自定义字段');
  if(Number($('#row-limit').value)!==1000)parts.push(`最多 ${Number($('#row-limit').value)||1000} 行`);
  const text=parts.length?parts.join(' · '):'显示原始数据';
  $('#query-summary').textContent=text;
  $('#builder-state').textContent=parts.length?`${parts.length} 个步骤`:'原始数据';
}
function configureJoinBuilder(){
  if(!active)return;
  const picker=$('#join-table');
  const candidates=tables.filter(t=>key(t)!==key(active));
  picker.innerHTML='<option value="">不关联</option>'+candidates.map(t=>`<option value="${esc(key(t))}">${esc(t.schema+'.'+t.name)}</option>`).join('');
  $('#join-left-field').innerHTML=optionList(active.columns);
  $('#join-right-field').innerHTML='<option value="">请先选择关联表</option>';
  $('#join-fields').textContent='请先选择关联表。';
  $('#join-hint').textContent='选择关联表后，指定两边用于匹配的字段；系统只会生成安全的只读查询。';
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
  $('#join-fields').innerHTML=target.columns.map(c=>`<label><input type="checkbox" value="${esc(c.name)}"> ${esc(c.name)} <small>${esc(c.data_type||'')}</small></label>`).join('');
  $('#join-hint').textContent=relation?'已根据外键元数据预选关联字段。可按需要调整。':'未发现外键标记，请选择两边实际对应的字段。';
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
  });
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
  try{
    const d=await api('/api/dataset','POST',query);
    lastQueryIR=d.queryir||query;
    lastChart=d.chartspec;
    renderGrid(d,'visual');
  }catch(e){
    $('#grid-status').textContent='筛选失败：'+e.message;
    toast(e.message);
  }
}
function renderGrid(d, mode){
  mode=mode||lastResultMode||'visual';
  lastResultMode=mode;
  const cols=d.columns||[];
  const rows=d.rows||[];
  TopbaseCode.setCode('#generated-sql',d.sql||'',{language:'sql',label:'本次执行的 SQL'});
  queryEditor.setGeneratedSQL(d.sql||'');
  const n=mode==='visual'?(lastQueryIR&&lastQueryIR.filters||[]).length:0;
  $('#grid-status').textContent=`数据库返回 ${rows.length} 行`+(mode==='sql'?'（SQL 实时查询）':(n?`（已应用 ${n} 条筛选）`:'。可在上方添加查询步骤。'));
  if(!cols.length){$('#grid-wrap').innerHTML='<p>这张表没有返回列。</p>';return}
  const types={},descriptions={};
  tables.forEach(table=>(table.columns||[]).forEach(column=>{
    if(types[column.name]===undefined)types[column.name]=column.data_type;
    if(descriptions[column.name]===undefined)descriptions[column.name]=column.description||'';
  }));
  fieldMeta.forEach(field=>{if(field.description)descriptions[field.name]=field.description});
  TopbaseGrid('#grid-wrap',{columns:cols, rows, aliases, types, descriptions, filtersEnabled:false, hidden:gridState.hidden, onChange:state=>{gridState.hidden=state.hidden}});
}
queryEditor=TopbaseQueryEditor.mount('#ask-panel',{
  onSQLChange:()=>{lastChart=null},
  onModeChange:mode=>{
    $$('.visual-result-action').forEach(item=>item.hidden=mode==='sql');
    if(mode==='sql'){
      queryEditor&&queryEditor.setSummary('执行自定义 SQL');
      $('#builder-state').textContent='SQL 模式';
    }else{
      updateBuilderSummary();
    }
  },
  onRun:mode=>mode==='sql'?runNativeSQL():runVisualQuery()
});
TopbaseCode.mountBlock('#generated-sql',{language:'sql',label:'本次执行的 SQL'});
$('#back-dbs').onclick=()=>creationMode?location.assign('/questions/new/'):showDatabases();
$('#db-search').oninput=renderDatabases;
$('#search').oninput=e=>renderTables(tables.filter(t=>(t.name+' '+t.schema+' '+(t.description||'')+' '+(t.columns||[]).map(c=>c.name+' '+(c.description||'')).join(' ')).toLowerCase().includes(e.target.value.toLowerCase())));
$('#toggle-ask').onclick=()=>{
  const panel=$('#ask-panel');
  panel.hidden=!panel.hidden;
  $('#toggle-ask').textContent=panel.hidden?'展开查询编辑器':'收起查询编辑器';
  $('#toggle-ask').setAttribute('aria-expanded',panel.hidden?'false':'true');
};
$('#join-table').onchange=updateJoinTarget;
$('#fields').onchange=()=>{updateSelectedFieldCount();updateBuilderSummary()};
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
  try{
    const d=await api('/api/dataset','POST',query);
    lastQueryIR=d.queryir||query;
    lastChart=d.chartspec;
    renderGrid(d,'visual');updateBuilderSummary();toast('查询已完成');
  }catch(e){$('#grid-status').textContent='查询失败：'+e.message;toast(e.message)}
}
async function runNativeSQL(){
  if(!active||!currentDB)return;
  const sql=queryEditor.sql().trim();
  if(!sql){toast('请先输入要执行的 SQL');return}
  $('#grid-status').textContent='正在执行 SQL…';
  try{
    const d=await api('/api/queries/run','POST',{database_id:currentDB.id,sql});
    lastChart=null;
    renderGrid(Object.assign({},d,{sql:d.sql||sql}),'sql');
    toast('SQL 查询已完成');
  }catch(e){$('#grid-status').textContent='SQL 查询失败：'+e.message;toast(e.message)}
}
$('#save-question').onclick=async()=>{
  if(!lastQueryIR&&!queryEditor.sql().trim())return toast('请先打开一张表并完成查询');
  const name=await promptDialog({kicker:'保存查询',title:'将当前查询保存为分析',label:'分析名称',value:(active&&($('#table-name').textContent||active.name))||'未命名分析',placeholder:'例如：活跃客户明细',confirmText:'保存分析'});
  if(!name)return;
  const sql=queryEditor.sql().trim();
  const payload=queryEditor.mode()==='sql'
    ?{name,query_type:'native',database_id:currentDB.id,native_sql:sql,chartspec:lastChart}
    :{name,query_type:'queryir',queryir:lastQueryIR,chartspec:lastChart};
  if(payload.query_type==='native'&&!sql)return toast('请先输入要保存的 SQL');
  try{const saved=await api('/api/questions','POST',payload);toast('已保存为分析');location.href='/questions/'+saved.id+'/'}catch(e){toast(e.message)}
};
async function drill(kind){
  if(queryEditor.mode()==='sql')return toast('SQL 结果暂不支持可视化下钻，请切换到可视化查询。');
  if(!lastQueryIR)return toast('请先查看表数据');
  const field=$('#drill-field').value|| (lastQueryIR.filters&&lastQueryIR.filters[0]&&lastQueryIR.filters[0].field) || (lastQueryIR.group_by&&lastQueryIR.group_by[0]&&lastQueryIR.group_by[0].field);
  const value=(lastQueryIR.filters&&lastQueryIR.filters[0]&&lastQueryIR.filters[0].value)||'';
  try{
    const selected=tableForJoin($('#join-table').value);
    const d=await api('/api/dataset/drill','POST',{queryir:lastQueryIR,drill:{kind,field,value},join_table:selected?selected.name:''});
    if(d.queryir) lastQueryIR=d.queryir;
    if(filterBar) filterBar.setFilters(lastQueryIR.filters||[]);
    renderGrid(d,'visual');toast('下钻完成');
  }catch(e){toast(e.message)}
}
$('#drill-records').onclick=()=>drill('records');
$('#drill-filter').onclick=()=>drill('filter');
async function boot(){
  try{
    const me=await api('/api/user/current').catch(()=>null);
    isAdmin=!!(me&&me.is_admin);
    databases=await api('/api/databases');
    const params=new URLSearchParams(location.search);
    creationMode=params.get('from')==='new-analysis';
    if(creationMode){$('#section-title').textContent='新建分析';$('#back-dbs').textContent='← 更换起始数据';$('#toggle-ask').textContent='查询步骤'}
    const db=params.get('db') || topbasePickDatabase(databases);
    renderDatabases();
    if(db && databases.some(d=>d.id===db)){
      topbaseRememberDatabase(db);
      await openDatabase(db);
      const schema=params.get('schema'), table=params.get('table');
      if(schema&&table){
        const found=tables.find(t=>t.schema===schema&&t.name===table);
        if(found) await openTable(found);
      }
    }
  }catch(e){toast(e.message)}
}
boot();
