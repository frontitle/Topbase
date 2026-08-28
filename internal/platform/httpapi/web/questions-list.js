// Analysis list and grouping surface.
let questions=[], collections=[], user=null, permissionGroups=[];
function typeLabel(question){return question.query_type==='native'?'SQL':'可视化'}
function collectionName(id){const collection=collections.find(item=>item.id===id);return collection?collection.name:'未分组'}
function collectionKind(collection){return collection.read_only?(collection.shared_by_name||'其他成员')+' 共享 · 仅查看':(collection.kind==='personal_project'?'个人分组':'企业项目')}
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
  $('#group-list').innerHTML=collections.map(collection=>{const shared=!!collection.read_only;return `<article class="analysis-group-card ${shared?'shared':''}"><div><b>${esc(collection.name)}</b><small>${collectionKind(collection)} · ${analysisCount(collection.id)} 条分析 · ${childCount(collection.id)} 个子分组</small></div><footer><span>${shared?'共享分组仅可查看':'在详情中整理、移动或共享内容'}</span><a class="secondary" href="/collections/${encodeURIComponent(collection.id)}/">${shared?'查看分组':'配置分组'}</a></footer></article>`}).join('')||emptyHTML({icon:'☰',title:'还没有分组',body:'创建个人分组或企业项目后，可以把相关分析整理在一起。'});
}
async function createGroup(){
  const admin=!!(user&&user.is_admin);
  const fields=[{name:'name',label:'分组名称',placeholder:'例如：经营分析',required:true}];
  if(admin){
    fields.push({name:'kind',label:'分组位置',type:'choice',value:'personal_project',required:true,options:[{value:'personal_project',label:'个人分组',description:'由你整理，可共享给其他成员查看。'},{value:'team_project',label:'企业项目',description:'团队成员角色由管理后台统一配置。'}]});
    if(permissionGroups.length)fields.push({name:'owner_group_id',label:'初始管理用户组',type:'select',value:permissionGroups[0].id,help:'仅企业项目使用；之后可在管理后台配置各用户组角色。',options:permissionGroups.map(group=>({value:group.id,label:group.name}))});
  }
  const values=await formDialog({kicker:'分析分组',title:'新建分组',description:'分组只用于整理内容；企业项目的成员角色统一在管理后台设置。',confirmText:'创建分组',fields,validate:value=>admin&&value.kind==='team_project'&&permissionGroups.length&&!value.owner_group_id?'请选择企业项目的初始管理用户组。':''});
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
  $('#collection').innerHTML='<option value="">全部分组</option>'+collections.map(collection=>'<option value="'+esc(collection.id)+'">'+esc(collection.name)+'</option>').join('');
  renderAnalyses();renderGroups();
}
$('#search').oninput=renderAnalyses;$('#collection').onchange=renderAnalyses;
$('#create-group').onclick=createGroup;
load().catch(error=>{$('#list').innerHTML=emptyHTML({icon:'!',title:'分析列表暂时无法加载',body:'请刷新页面后重试；若问题持续存在，请检查登录状态和分析权限。'});toast(error.message)});
