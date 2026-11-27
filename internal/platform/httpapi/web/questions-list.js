// Analysis list and grouping surface.
let questions=[], collections=[], user=null, permissionGroups=[];
function typeLabel(question){return question.query_type==='native'?'SQL':'可视化'}
function collectionName(id){const collection=collections.find(item=>item.id===id);return collection?collection.name:TopbaseLocale.t('未分组')}
function collectionKind(collection){return collection.read_only?TopbaseLocale.t('{name} 共享 · 仅查看',{name:collection.shared_by_name||TopbaseLocale.t('其他成员')}):TopbaseLocale.t(collection.kind==='personal_project'?'个人分组':'企业项目')}
function analysisCount(id){return questions.filter(question=>question.collection_id===id).length}
function childCount(id){return collections.filter(collection=>collection.parent_id===id).length}
function renderAnalyses(){
  const query=($('#search').value||'').toLowerCase();
  const collection=$('#collection').value;
  const items=questions.filter(item=>(!collection||item.collection_id===collection)&&((item.name||'')+' '+(item.description||'')).toLowerCase().includes(query));
  $('#list').innerHTML=items.map(item=>cardHTML({href:'/questions/'+item.id+'/',title:item.name,metaTranslate:false,meta:TopbaseLocale.t(typeLabel(item))+' · '+collectionName(item.collection_id)+' · '+new Date(item.created_at).toLocaleString(TopbaseLocale.locale())})).join('')||emptyHTML({icon:'◇',title:'还没有分析',body:'选择数据表，创建第一条分析。',href:'/questions/new/',cta:'新建分析'});
}
function renderGroups(){
  $('#group-count').textContent=collections.length?'('+collections.length+')':'';
  const groups = collections.map(collection => {
    const shared = !!collection.read_only;
    const counts = TopbaseLocale.t('{analyses} 条分析 · {groups} 个子分组', {
      analyses: analysisCount(collection.id), groups: childCount(collection.id)
    });
    return `<article class="analysis-group-card ${shared ? 'shared' : ''}">
      <div><b translate="no">${esc(collection.name)}</b>
        <small translate="no">${esc(collectionKind(collection))} · ${esc(counts)}</small></div>
      <footer><span>${shared ? '共享分组仅可查看' : '整理分组内容'}</span>
        <a class="secondary tb-btn tb-btn-secondary" href="/collections/${encodeURIComponent(collection.id)}/">${shared ? '查看分组' : '配置分组'}</a></footer>
    </article>`;
  });
  $('#group-list').innerHTML = groups.join('') || emptyHTML({icon:'☰',title:'暂无分组',body:'创建分组，整理相关分析。'});
}
async function createGroup(){
  const admin=!!(user&&user.is_admin);
  const fields=[{name:'name',label:'分组名称',placeholder:'例如：经营分析',required:true}];
  if(admin){
    fields.push({name:'kind',label:'分组位置',type:'choice',value:'personal_project',required:true,options:[{value:'personal_project',label:'个人分组',description:'由你整理，可共享给其他成员查看。'},{value:'team_project',label:'企业项目',description:'团队成员角色由管理后台统一配置。'}]});
    if(permissionGroups.length)fields.push({name:'owner_group_id',label:'初始管理用户组',type:'select',value:permissionGroups[0].id,help:'用于企业项目，可在管理后台更改。',options:permissionGroups.map(group=>({value:group.id,label:group.name,translate:false}))});
  }
  const values=await formDialog({kicker:'分析分组',title:'新建分组',description:'个人分组可供成员查看，企业项目的成员角色由管理员设置。',confirmText:'创建分组',fields,validate:value=>admin&&value.kind==='team_project'&&permissionGroups.length&&!value.owner_group_id?'请选择企业项目的初始管理用户组。':''});
  if(!values)return;
  const kind=admin?values.kind:'personal_project';
  try{
    await api('/api/collections','POST',{name:values.name,kind,owner_group_id:kind==='team_project'?values.owner_group_id:''});
    toast('分组已创建');await load();document.querySelector('#analysis-groups').scrollIntoView({behavior:'smooth',block:'start'});
  }catch(error){toast(error.message)}
}
async function load(){
  // The analysis list must not disappear just because a supplemental request fails.
  questions=await api('/api/questions');
  [collections,user]=await Promise.all([api('/api/collections').catch(()=>[]),api('/api/user/current').catch(()=>null)]);
  permissionGroups=user&&user.is_admin?await api('/api/groups').catch(()=>[]):[];
  $('#collection').innerHTML='<option value="">全部分组</option>'+collections.map(collection=>'<option translate="no" value="'+esc(collection.id)+'">'+esc(collection.name)+'</option>').join('');
  renderAnalyses();renderGroups();
}
$('#search').oninput=renderAnalyses;$('#collection').onchange=renderAnalyses;
$('#create-group').onclick=createGroup;
load().catch(error=>{$('#list').innerHTML=emptyHTML({icon:'!',title:'分析列表暂时无法加载',body:'请刷新重试，或联系管理员检查权限。'});toast(error.message)});
