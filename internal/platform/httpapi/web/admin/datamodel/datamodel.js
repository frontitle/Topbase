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
  $('#table-list').innerHTML=items.map(t=>`<div class="table-item-row ${active&&key(active)===key(t)?'active':''}"><button class="table-item" data-key="${esc(key(t))}" title="${esc(t.display_name||t.name)}"><b>${esc(t.display_name||t.name)}</b><small title="${esc(t.user_note||t.description||'')}">${esc(t.user_note||t.description||'暂无说明')}</small></button><button class="table-visibility-icon ${t.hidden?'is-hidden':''}" data-toggle-hidden="${esc(key(t))}" type="button" title="${t.hidden?'在前台显示':'在前台隐藏'}" aria-label="${t.hidden?'在前台显示':'在前台隐藏'}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"></path><circle cx="12" cy="12" r="2.8"></circle>${t.hidden?'<path d="M4 4 20 20"></path>':''}</svg></button></div>`).join('')||'<p>没有表。请先在数据源里同步结构。</p>';
  $$('#table-list .table-item').forEach(b=>b.onclick=()=>openTable(tables.find(t=>key(t)===b.dataset.key)).catch(e=>toast(e.message)));
  $$('[data-toggle-hidden]').forEach(b=>b.onclick=async event=>{event.stopPropagation();const table=tables.find(t=>key(t)===b.dataset.toggleHidden);if(!table)return;await saveAnnotation(table,!table.hidden);});
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
  renderTables(q?tables.filter(t=>((t.display_name||'')+' '+t.name+' '+t.schema).toLowerCase().includes(q)):tables);
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
  table.hidden=!!note.hidden;setEditorVisibility();
  $('#source-description').textContent=table.description||'数据源没有提供表注释';$('#source-description').title=table.description||'数据源没有提供表注释';
  $('#description').value=note.user_note||'';
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
      <td><button class="field-visibility-icon ${f.visibility==='hidden'?'is-hidden':''}" data-k="visibility" type="button" title="${f.visibility==='hidden'?'在前台显示':'在前台隐藏'}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"></path><circle cx="12" cy="12" r="2.8"></circle>${f.visibility==='hidden'?'<path d="M4 4 20 20"></path>':''}</svg></button></td>
    </tr>`;
  }).join('');
  $$('#field-body tr').forEach(row=>{
    const f=byName[row.dataset.field]||{};
    const c=(table.columns||[]).find(column=>column.name===row.dataset.field)||{};
    row.querySelector('[data-k="semantic"]').value=f.semantic_type||(c.primary_key?'EntityKey':(c.foreign_key?'ForeignKey':''));
    let fieldTimer;row.querySelectorAll('input').forEach(input=>input.oninput=()=>{clearTimeout(fieldTimer);fieldTimer=setTimeout(()=>saveField(row),500)});row.querySelector('select').onchange=()=>saveField(row);row.querySelector('[data-k="visibility"]').onclick=()=>{const button=row.querySelector('[data-k="visibility"]'),hidden=button.classList.toggle('is-hidden');button.title=hidden?'在前台显示':'在前台隐藏';button.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"></path><circle cx="12" cy="12" r="2.8"></circle>'+(hidden?'<path d="M4 4 20 20"></path>':'')+'</svg>';saveField(row)};
  });
}
$('#database').onchange=()=>{topbaseRememberDatabase($('#database').value);history.replaceState({},'','/admin/datamodel/?db='+encodeURIComponent($('#database').value));loadTables().catch(e=>toast(e.message))};
$('#search').oninput=()=>{
  const q=($('#search').value||'').toLowerCase();
  renderTables(q?tables.filter(t=>((t.display_name||'')+' '+t.name+' '+t.schema).toLowerCase().includes(q)):tables);
};
let saveTimer;
function setEditorVisibility(){const button=$('#editor-visibility');if(!button||!active)return;button.classList.toggle('is-hidden',!!active.hidden);button.title=active.hidden?'在前台显示':'在前台隐藏';button.setAttribute('aria-label',button.title);button.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"></path><circle cx="12" cy="12" r="2.8"></circle>'+(active.hidden?'<path d="M4 4 20 20"></path>':'')+'</svg>'}
async function saveAnnotation(table=active, hidden=active&&active.hidden){
  if(!table)return;
  const id=$('#database').value;
  try{
    await api(`/api/databases/${id}/tables/${encodeURIComponent(table.schema)}/${encodeURIComponent(table.name)}/annotation`,'PUT',{
      display_name:table===active?$('#display-name').value.trim():(table.display_name||''), description:'', user_note:table===active?$('#description').value.trim():'', hidden, field_types:{}
    });
    table.hidden=hidden;if(table===active){table.display_name=$('#display-name').value.trim();$('#editor-title').textContent=table.display_name||table.name;setEditorVisibility();}
    renderTables(tables);toast(hidden?'已在前台隐藏':'已恢复前台可见');
  }catch(e){toast(e.message)}
}
function scheduleAnnotationSave(){clearTimeout(saveTimer);saveTimer=setTimeout(()=>saveAnnotation(),500)}
async function saveField(row){if(!active)return;const id=$('#database').value,name=row.dataset.field,semantic=row.querySelector('[data-k="semantic"]').value,fkTable=row.querySelector('[data-k="fk_table"]').value.trim(),fkField=row.querySelector('[data-k="fk_field"]').value.trim(),visibility=row.querySelector('[data-k="visibility"]').classList.contains('is-hidden')?'hidden':'';const payload={name,display_name:row.querySelector('[data-k="display_name"]').value.trim(),description:row.querySelector('[data-k="description"]').value.trim(),semantic_type:semantic,visibility};if(fkTable&&fkField){const source=(active.columns||[]).find(c=>c.name===name),foreign=source&&source.foreign_key;payload.semantic_type=payload.semantic_type||'ForeignKey';payload.fk_target={schema:foreign&&foreign.table===fkTable?foreign.schema:active.schema,table:fkTable,name:fkField}}try{await api(`/api/databases/${id}/tables/${encodeURIComponent(active.schema)}/${encodeURIComponent(active.name)}/fields`,'PUT',payload);toast('字段已自动保存')}catch(e){toast(e.message)}}
$('#display-name').oninput=scheduleAnnotationSave;$('#description').oninput=scheduleAnnotationSave;$('#editor-visibility').onclick=()=>{if(active)saveAnnotation(active,!active.hidden)};
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
