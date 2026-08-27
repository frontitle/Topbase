let types=[], tables=[], active=null;
function key(t){return t.schema+'.'+t.name}
function params(){return new URLSearchParams(location.search)}
function setURL(){
  const p=new URLSearchParams();
  const id=$('#database').value;
  if(id) p.set('db', id);
  if(active){p.set('schema', active.schema);p.set('table', active.name)}
  history.replaceState({}, '', '/admin/datamodel/'+(p.toString()?'?'+p:''));
}
function renderTables(items){
  $('#table-count').textContent=items.length;
  $('#table-list').innerHTML=items.map(t=>`<button class="table-item ${active&&key(active)===key(t)?'active':''}" data-key="${esc(key(t))}" title="${esc(t.description||'')}"><b>${esc(t.schema)}.${esc(t.name)}</b><small>${esc(t.description||((t.columns||[]).length+' 个字段'))}</small></button>`).join('')||'<p>没有表。请先在数据源里同步结构。</p>';
  $$('#table-list .table-item').forEach(b=>b.onclick=()=>openTable(tables.find(t=>key(t)===b.dataset.key)).catch(e=>toast(e.message)));
}
async function loadTables(){
  const id=$('#database').value;
  active=null;$('#meta-editor').hidden=true;$('#empty-editor').hidden=false;
  if(!id){$('#table-list').textContent='先选择数据源。';return}
  try{tables=await api('/api/databases/'+id+'/tables')}catch(_){
    try{tables=await api('/api/databases/'+id+'/tables?cached=1')}catch(e){toast(e.message);tables=[]}
  }
  renderTables(tables);
  setURL();
  const schema=params().get('schema'), table=params().get('table');
  if(schema&&table){
    const found=tables.find(t=>t.schema===schema&&t.name===table);
    if(found) await openTable(found);
  }
}
async function openTable(table){
  active=table;
  const q=($('#search').value||'').toLowerCase();
  renderTables(q?tables.filter(t=>(t.name+' '+t.schema).toLowerCase().includes(q)):tables);
  $('#empty-editor').hidden=true;$('#meta-editor').hidden=false;
  $('#physical-name').textContent=table.schema+'.'+table.name;
  $('#editor-title').textContent=table.name;
  setURL();
  const id=$('#database').value;
  const [note, saved]=await Promise.all([
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/annotation`).catch(()=>({})),
    api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/fields`).catch(()=>[])
  ]);
  $('#display-name').value=note.display_name||'';
  $('#description').value=note.description||table.description||'';
  const byName=Object.fromEntries((saved||[]).map(f=>[f.name,f]));
  const typeOpts='<option value="">未设置</option>'+types.map(t=>`<option value="${t}">${t}</option>`).join('');
  $('#field-body').innerHTML=(table.columns||[]).map(c=>{
    const f=byName[c.name]||{};
    const foreign=f.fk_target||c.foreign_key||{};
    const description=f.description||c.description||'';
    const semantic=f.semantic_type||(c.primary_key?'EntityKey':(c.foreign_key?'ForeignKey':''));
    return `<tr data-field="${esc(c.name)}" data-source-fk-schema="${esc(c.foreign_key&&c.foreign_key.schema||'')}">
      <td><b>${esc(c.name)}</b>${c.primary_key?'<small class="field-badge">主键</small>':''}</td>
      <td><input data-k="display_name" placeholder="别称" value="${esc(f.display_name||'')}"></td>
      <td><span>${esc(c.data_type||'')}</span>${c.default_value?`<small class="field-default" title="${esc(c.default_value)}">默认：${esc(c.default_value)}</small>`:''}</td>
      <td><input data-k="description" placeholder="数据库注释或业务说明" value="${esc(description)}"></td>
      <td><select data-k="semantic">${typeOpts}</select></td>
      <td><input data-k="fk_table" placeholder="users" value="${esc(foreign.table||'')}"></td>
      <td><input data-k="fk_field" placeholder="id" value="${esc(foreign.name||'')}"></td>
    </tr>`;
  }).join('');
  $$('#field-body tr').forEach(row=>{
    const f=byName[row.dataset.field]||{};
    const c=(table.columns||[]).find(column=>column.name===row.dataset.field)||{};
    row.querySelector('[data-k="semantic"]').value=f.semantic_type||(c.primary_key?'EntityKey':(c.foreign_key?'ForeignKey':''));
  });
}
$('#database').onchange=()=>{topbaseRememberDatabase($('#database').value);history.replaceState({},'','/admin/datamodel/?db='+encodeURIComponent($('#database').value));loadTables().catch(e=>toast(e.message))};
$('#search').oninput=()=>{
  const q=($('#search').value||'').toLowerCase();
  renderTables(q?tables.filter(t=>(t.name+' '+t.schema).toLowerCase().includes(q)):tables);
};
$('#save-all').onclick=async()=>{
  if(!active)return;
  const id=$('#database').value;
  const schema=active.schema, table=active.name;
  try{
    await api(`/api/databases/${id}/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}/annotation`,'PUT',{
      display_name:$('#display-name').value, description:$('#description').value, field_types:{}
    });
    const rows=$$('#field-body tr');
    for(const row of rows){
      const name=row.dataset.field;
      const semantic=row.querySelector('[data-k="semantic"]').value;
      const fkTable=row.querySelector('[data-k="fk_table"]').value.trim();
      const fkField=row.querySelector('[data-k="fk_field"]').value.trim();
      const payload={
        name,
        display_name:row.querySelector('[data-k="display_name"]').value.trim(),
        description:row.querySelector('[data-k="description"]').value.trim(),
        semantic_type:semantic
      };
      if(fkTable&&fkField){
        const sourceColumn=(active.columns||[]).find(column=>column.name===name);
        const sourceForeign=sourceColumn&&sourceColumn.foreign_key;
        const targetSchema=sourceForeign&&sourceForeign.table===fkTable?sourceForeign.schema:schema;
        payload.semantic_type=payload.semantic_type||'ForeignKey';
        payload.fk_target={schema:targetSchema||schema, table:fkTable, name:fkField};
      }
      await api(`/api/databases/${id}/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}/fields`,'PUT',payload);
    }
    $('#editor-title').textContent=$('#display-name').value.trim()||table;
    toast('已保存表与字段元数据');
  }catch(e){toast(e.message)}
};
async function boot(){
  types=await api('/api/semantic-types');
  const dbs=await api('/api/databases');
  const selected=topbasePickDatabase(dbs, params().get('db')||'');
  $('#database').innerHTML=(dbs.length?dbs.map(d=>`<option value="${esc(d.id)}">${esc(d.name)}</option>`):'<option value="">暂无数据源</option>').join('');
  if(selected) $('#database').value=selected;
  if($('#database').value){
    topbaseRememberDatabase($('#database').value);
    await loadTables();
  }
}
boot().catch(e=>toast(e.message));
