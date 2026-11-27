let questions=[];
const frequencyCron={hourly:'0 * * * *',daily:'0 9 * * *',weekly:'0 9 * * 1'};
function slug(value){return String(value||'').toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_+|_+$/g,'').slice(0,48)||'analysis_result'}
function suggestedTarget(question) {
  const suffix = Array.from(crypto.getRandomValues(new Uint8Array(6)), byte => byte.toString(16).padStart(2, '0')).join('');
  return slug(question.name).slice(0, 35) + '_' + suffix;
}
function tableURL(table){return '/data/?db='+encodeURIComponent(table.database_id)+'&schema='+encodeURIComponent(table.schema)+'&table='+encodeURIComponent(table.name)}
function scheduleLabel(schedule){if(!schedule)return '尚未配置';if(schedule.cron==='0 * * * *')return '每小时';if(schedule.cron==='0 9 * * 1')return '每周一 09:00';if(schedule.cron==='0 9 * * *')return '每天 09:00';return schedule.cron}
const statusLabels = {succeeded:'更新成功', failed:'更新失败', running:'更新中', writing:'等待写入确认', needs_review:'待管理员核对', pending:'待首次更新'};
const strategyLabels = {replace:'全量刷新', incremental:'增量更新', truncate_insert:'清空后写入', create_only:'仅创建'};
function materializationRow(schedule, table, question) {
  const target = table ? table.schema + '.' + table.name : schedule.materialize_to;
  const title = schedule?.name || target;
  const result = table ? TopbaseLocale.t('已保存 {count} 行', {count: (table.row_count || 0).toLocaleString(TopbaseLocale.locale())}) : '尚无可用结果';
  const status = schedule?.last_status || table?.last_status || 'pending';
  const statusClass = status === 'failed' ? 'failed' : status === 'succeeded' ? '' : 'pending';
  const lastAttempt = schedule?.last_run_at || table?.last_run_at;
  const lastUpdate = lastAttempt ? new Date(lastAttempt).toLocaleString(TopbaseLocale.locale()) : '尚未更新';
  const strategy = schedule ? strategyLabels[schedule.strategy || 'replace'] || schedule.strategy : '—';
  const runButton = schedule ? '<button type="button" class="secondary tb-btn tb-btn-secondary" data-run="' + esc(schedule.id) + '">' +
    (status === 'failed' ? '重试更新' : '立即更新') + '</button>' : '';
  const analyzeLink = table ? '<a class="secondary tb-btn tb-btn-secondary" href="' + esc(tableURL(table)) + '">创建分析</a>' : '';
  return `<tr>
    <td><b translate="no">${esc(title)}</b><small><span translate="no">${esc(target)}</span> · ${esc(result)}</small>
      ${table?.watermark ? '<small>' + esc(TopbaseLocale.t('水位 {value}', {value: table.watermark})) + '</small>' : ''}</td>
    <td><b ${question ? 'translate="no"' : ''}>${esc(question?.name || (schedule?.model_id ? '来源模型' : '原始分析已不可用'))}</b>
      ${question ? '<small>' + (question.query_type === 'native' ? 'SQL 分析' : '可视化分析') + '</small>' : ''}</td>
    <td><b>${esc(scheduleLabel(schedule))}</b><small>${esc(schedule?.timezone || '')}</small></td>
    <td><b>${esc(strategy)}</b><small translate="no">${esc(schedule?.watermark_field || '')}</small></td>
    <td>${esc(lastUpdate)}</td>
    <td><span class="warehouse-status ${statusClass}">${esc(statusLabels[status] || status)}</span>
      ${schedule?.last_error ? '<small class="warehouse-error">' + esc(warehouseErrorText(schedule.last_error)) + '</small>' : ''}</td>
    <td><div class="warehouse-table-actions">${analyzeLink}${runButton}</div></td>
  </tr>`;
}

let currentLoad = 0;
async function load(){
  const generation = ++currentLoad;
  const [tables,schedules,items]=await Promise.all([api('/api/warehouse/tables'),api('/api/schedules?include=last_run'),api('/api/questions')]);
  if (generation !== currentLoad) return;
  questions=items;
  const schedulesByID = new Map(schedules.map(plan => [plan.id, plan]));
  const questionsByID = new Map(items.map(item => [item.id, item]));
  const rows = tables.map(table => {
    const plan = schedulesByID.get(table.schedule_id);
    return materializationRow(plan, table, questionsByID.get(plan?.question_id || table.question_id));
  });
  const produced = new Set(tables.map(table => table.schedule_id));
  const plansWithoutResults = schedules.filter(plan => !produced.has(plan.id));
  const pending = plansWithoutResults.map(plan => materializationRow(plan, null, questionsByID.get(plan.question_id)));
  $('#materialization-count').textContent = TopbaseLocale.t('{count} 项', {count: pending.length + rows.length});
  $('#materializations').innerHTML = pending.concat(rows).join('') || '<tr><td class="warehouse-empty" colspan="7">暂无沉淀数据。点击“新建数据沉淀”并选择分析。</td></tr>';
  $$('[data-run]').forEach(button => {
    button.onclick = async () => {
      const label = button.textContent;
      button.disabled = true;
      button.textContent = '更新中…';
      try {
        await updateMaterialization(button.dataset.run, false);
      } finally {
        button.disabled = false;
        button.textContent = label;
      }
    };
  });
}
async function updateMaterialization(scheduleID, firstRun) {
  try {
    const run = await api('/api/schedules/'+encodeURIComponent(scheduleID)+'/run','POST',{});
    toast(TopbaseLocale.t('已沉淀 {count} 行', {count: run.row_count}));
  } catch (error) {
    toast(TopbaseLocale.t(firstRun ? '计划已创建，首次更新失败。{error}' : '更新失败。{error}', {error: warehouseErrorText(error.message)}));
  }
  await load().catch(error => toast(TopbaseLocale.t('列表刷新失败。{error}', {error: warehouseErrorText(error.message)})));
}
async function createMaterialization(){
  if(!questions.length){toast('请先创建一条分析，再进行数据沉淀。');return}
  const wanted=new URLSearchParams(location.search).get('question');
  const selected=questions.find(question=>question.id===wanted)||questions[0];
  const values=await formDialog({
    kicker:'数据沉淀',title:'新建数据沉淀',description:'创建后立即保存结果，随后按计划更新。',confirmText:'创建并立即沉淀',size:'wide',
    fields:[
      {name:'question_id',label:'来源分析',type:'select',value:selected.id,required:true,options:questions.map(question=>({value:question.id,label:question.name+' · '+(question.query_type==='native'?'SQL':TopbaseLocale.t('可视化')),translate:false})),help:'复用已验证的分析逻辑。'},
      {name:'name',label:'沉淀名称',value:TopbaseLocale.t('{name} 数据', {name: selected.name}),required:true,placeholder:'例如：每日订单明细'},
      {name:'frequency',label:'更新频率',type:'select',value:'daily',required:true,options:[{value:'hourly',label:'每小时'},{value:'daily',label:'每天 09:00'},{value:'weekly',label:'每周一 09:00'}]},
      {name:'target',label:'保存表名',value:suggestedTarget(selected),required:true,placeholder:'例如：daily_orders',help:'使用字母、数字或下划线，保存到 warehouse.wh_*。'},
      {name:'strategy',label:'保存方式',type:'select',value:'replace',required:true,options:[{value:'replace',label:'全量刷新'},{value:'incremental',label:'仅追加变化数据'}]},
      {name:'watermark_field',label:'水位字段（仅增量更新）',placeholder:'例如：created_at',help:'每次只读取此字段大于上次水位的记录。'}
    ],
    validate:value=>{
      if (!/^[a-zA-Z0-9_]{1,60}$/.test(value.target)) return '表名请使用 1 至 60 个字母、数字或下划线。';
      return value.strategy==='incremental'&&!value.watermark_field?'增量更新需要填写变化字段。':'';
    }
  });
  if(!values)return;
  try{
    const schedule=await api('/api/schedules','POST',{name:values.name,question_id:values.question_id,cron:frequencyCron[values.frequency],timezone:'Asia/Shanghai',materialize_to:'warehouse.'+values.target.toLowerCase(),strategy:values.strategy,watermark_field:values.watermark_field});
    history.replaceState({},'',location.pathname);
    await updateMaterialization(schedule.id, true);
  }catch(error){toast(warehouseErrorText(error.message))}
}
$('#create-materialization').onclick=createMaterialization;
load().then(() => {
  if (location.hash === '#create-materialization' && new URLSearchParams(location.search).has('question')) {
    if (!questions.some(question => question.id === new URLSearchParams(location.search).get('question'))) {
      toast('来源分析已不可用，请重新选择分析。');
      return;
    }
    return createMaterialization();
  }
}).catch(error=>toast(warehouseErrorText(error.message)));
