// Analysis list and grouping surface.
let questions=[], collections=[], user=null, permissionGroups=[];
function typeLabel(question){return question.query_type==='native'?'SQL':'可视化'}
function collectionName(id){const collection=collections.find(item=>item.id===id);return collection?collection.name:'未分组'}
function collectionKind(collection){return collection.kind==='personal_project'?'个人分组':'团队分组'}
function analysisCount(id){return questions.filter(question=>question.collection_id===id).length}
function childCount(id){return collections.filter(collection=>collection.parent_id===id).length}
function renderAnalyses(){
  const query=($('#search').value||'').toLowerCase();
  const collection=$('#collection').value;
  const items=questions.filter(item=>(!collection||item.collection_id===collection)&&((item.name||'')+' '+(item.description||'')).toLowerCase().includes(query));
  $('#list').innerHTML=items.map(item=>cardHTML({href:'/questions/'+item.id+'/',title:item.name,meta:typeLabel(item)+' · '+collectionName(item.collection_id)+' · '+new Date(item.created_at).toLocaleString('zh-CN')})).join('')||emptyHTML({icon:'◇',title:'还没有分析',body:'从源数据选择一张表，通过可视化构建器创建第一条分析。',href:'/questions/new/',cta:'新建分析'});
}
function renderGroups(){
  $('#group-count').textContent=collections.length?'('+collections.length+')':'';
  $('#group-list').innerHTML=collections.map(collection=>`<article class="analysis-group-card"><div><b>${esc(collection.name)}</b><small>${collectionKind(collection)} · ${analysisCount(collection.id)} 条分析 · ${childCount(collection.id)} 个子分组</small></div><footer><span>用于整理内容与协作权限</span><a class="secondary" href="/collections/${encodeURIComponent(collection.id)}/">管理分组</a></footer></article>`).join('')||emptyHTML({icon:'☰',title:'还没有分组',body:'创建分组后，可以把相关分析整理在一起。'});
}
function setView(view, updateURL){
  const groups=view==='groups';
  $('#analysis-view').hidden=groups;
  $('#groups-view').hidden=!groups;
  $$('[data-analysis-view]').forEach(button=>{const active=button.dataset.analysisView===view;button.classList.toggle('active',active);button.setAttribute('aria-selected',active?'true':'false')});
  if(updateURL){const url=new URL(location.href);if(groups)url.searchParams.set('view','groups');else url.searchParams.delete('view');history.replaceState({},'',url)}
}
async function createGroup(){
  const admin=!!(user&&user.is_admin&&permissionGroups.length);
  const fields=[{name:'name',label:'分组名称',placeholder:'例如：经营分析',required:true}];
  if(admin){
    fields.push({name:'kind',label:'分组类型',type:'choice',value:'team_project',required:true,options:[{value:'team_project',label:'团队分组',description:'团队成员可以共同使用和维护'},{value:'personal_project',label:'个人分组',description:'仅用于整理自己的分析'}]});
    fields.push({name:'owner_group_id',label:'管理用户组',type:'select',value:permissionGroups[0].id,help:'创建团队分组时选择负责管理的用户组。',options:permissionGroups.map(group=>({value:group.id,label:group.name}))});
  }
  const values=await formDialog({kicker:'分析分组',title:'新建分组',description:'分组用于整理分析和管理协作权限，不会复制数据。',confirmText:'创建分组',fields,validate:value=>admin&&value.kind==='team_project'&&!value.owner_group_id?'团队分组必须选择管理用户组。':''});
  if(!values)return;
  const kind=admin?values.kind:'personal_project';
  try{
    await api('/api/collections','POST',{name:values.name,kind,owner_group_id:kind==='team_project'?values.owner_group_id:''});
    toast('分组已创建');await load();setView('groups',true);
  }catch(error){toast(error.message)}
}
async function load(){
  [questions,collections,user]=await Promise.all([api('/api/questions'),api('/api/collections'),api('/api/user/current').catch(()=>null)]);
  permissionGroups=user&&user.is_admin?await api('/api/groups').catch(()=>[]):[];
  $('#collection').innerHTML='<option value="">全部分组</option>'+collections.map(collection=>'<option value="'+esc(collection.id)+'">'+esc(collection.name)+'</option>').join('');
  renderAnalyses();renderGroups();
}
$('#search').oninput=renderAnalyses;$('#collection').onchange=renderAnalyses;
$$('[data-analysis-view]').forEach(button=>button.onclick=()=>setView(button.dataset.analysisView,true));
$('#create-group').onclick=createGroup;
setView(new URLSearchParams(location.search).get('view')==='groups'?'groups':'items');
load().catch(error=>toast(error.message));
