let questions=[];
const frequencyCron={hourly:'0 * * * *',daily:'0 9 * * *',weekly:'0 9 * * 1'};
function slug(value){return String(value||'').toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_+|_+$/g,'').slice(0,48)||'analysis_result'}
function tableURL(table){return '/data/?db='+encodeURIComponent(table.database_id)+'&schema='+encodeURIComponent(table.schema)+'&table='+encodeURIComponent(table.name)}
function scheduleLabel(schedule){if(!schedule)return '尚未配置';if(schedule.cron==='0 * * * *')return '每小时';if(schedule.cron==='0 9 * * 1')return '每周一 09:00';if(schedule.cron==='0 9 * * *')return '每天 09:00';return schedule.cron}
function status(table){const value=table.last_status||'pending';return {succeeded:'更新成功',failed:'更新失败',pending:'待首次更新'}[value]||value}
function statusClass(table){return table.last_status==='failed'?'failed':table.last_status==='succeeded'?'':'pending'}
async function load(){
  const [tables,schedules,items]=await Promise.all([api('/api/warehouse/tables'),api('/api/schedules'),api('/api/questions')]);
  questions=items;
  $('#materialization-count').textContent=tables.length+' 项';
  const byID=Object.fromEntries(schedules.map(schedule=>[schedule.id,schedule]));
  const questionsByID=Object.fromEntries(questions.map(question=>[question.id,question]));
  $('#materializations').innerHTML=tables.map(table=>{
    const schedule=byID[table.schedule_id], question=questionsByID[table.question_id];
    return '<tr><td><b>'+esc(table.schema+'.'+table.name)+'</b><small>'+(table.row_count||0)+' 行'+(table.watermark?' · 水位 '+esc(table.watermark):'')+'</small></td><td><b>'+esc(question&&question.name||'原始分析已不可用')+'</b><small>'+(question?(question.query_type==='native'?'SQL 分析':'可视化分析'):'—')+'</small></td><td><b>'+esc(scheduleLabel(schedule))+'</b><small>'+esc(schedule&&schedule.timezone||'Asia/Shanghai')+'</small></td><td><b>'+esc(schedule&&schedule.strategy==='incremental'?'增量更新':'全量刷新')+'</b><small>'+esc(schedule&&schedule.watermark_field||'每次重算完整结果')+'</small></td><td><b>'+esc(table.last_run_at?new Date(table.last_run_at).toLocaleString('zh-CN'):'尚未更新')+'</b></td><td><span class="warehouse-status '+statusClass(table)+'">'+esc(status(table))+'</span></td><td><div class="warehouse-table-actions"><a class="secondary" href="'+esc(tableURL(table))+'">创建分析</a>'+(schedule?'<button class="secondary" data-run="'+esc(schedule.id)+'" type="button">立即更新</button>':'')+'</div></td></tr>';
  }).join('')||'<tr><td class="warehouse-empty" colspan="7">还没有沉淀数据。点击右上角“新建数据沉淀”，选择一条分析后即可开始。</td></tr>';
  $$('[data-run]').forEach(button=>button.onclick=async()=>{button.disabled=true;button.textContent='更新中…';try{const run=await api('/api/schedules/'+button.dataset.run+'/run','POST',{});toast('已沉淀 '+run.row_count+' 行');await load()}catch(error){toast(error.message);button.disabled=false;button.textContent='立即更新'}});
}
async function createMaterialization(){
  if(!questions.length){toast('请先创建一条分析，再进行数据沉淀。');return}
  const wanted=new URLSearchParams(location.search).get('question');
  const selected=questions.find(question=>question.id===wanted)||questions[0];
  const values=await formDialog({
    kicker:'数据沉淀',title:'新建数据沉淀',description:'首次会立即把分析结果保存到本地表；后续按设置的计划更新。',confirmText:'创建并立即沉淀',size:'wide',
    fields:[
      {name:'question_id',label:'来源分析',type:'select',value:selected.id,required:true,options:questions.map(question=>({value:question.id,label:question.name+(question.query_type==='native'?' · SQL':' · 可视化')})),help:'复用已验证的分析逻辑。'},
      {name:'name',label:'沉淀名称',value:selected.name+' 数据',required:true,placeholder:'例如：每日订单明细'},
      {name:'frequency',label:'更新频率',type:'select',value:'daily',required:true,options:[{value:'hourly',label:'每小时'},{value:'daily',label:'每天 09:00'},{value:'weekly',label:'每周一 09:00'}]},
      {name:'target',label:'保存表名',value:slug(selected.name),required:true,placeholder:'例如：daily_orders',help:'会写入 warehouse.wh_*，不会改写源表。'},
      {name:'strategy',label:'保存方式',type:'select',value:'replace',required:true,options:[{value:'replace',label:'全量刷新'},{value:'incremental',label:'仅追加变化数据'}]},
      {name:'watermark_field',label:'变化字段（仅增量更新需要）',placeholder:'例如：created_at',help:'增量方式下，下一次只读取此字段晚于上次水位的数据。'}
    ],
    validate:value=>value.strategy==='incremental'&&!value.watermark_field?'增量更新需要填写变化字段。':''
  });
  if(!values)return;
  try{
    const schedule=await api('/api/schedules','POST',{name:values.name,question_id:values.question_id,cron:frequencyCron[values.frequency],timezone:'Asia/Shanghai',materialize_to:'warehouse.wh_'+slug(values.target),strategy:values.strategy,watermark_field:values.watermark_field});
    const run=await api('/api/schedules/'+schedule.id+'/run','POST',{});
    toast('数据沉淀已创建，首次已保存 '+run.row_count+' 行。');
    history.replaceState({},'',location.pathname);await load();
  }catch(error){toast(error.message)}
}
$('#create-materialization').onclick=createMaterialization;
load().catch(error=>toast(error.message));
