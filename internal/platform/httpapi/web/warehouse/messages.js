// Historical run errors are stored as plain text. Match only messages emitted by
// the warehouse service. Keep identifiers, watermarks and driver diagnostics intact.
function warehouseErrorText(message) {
  const text = String(message || '');
  if (!text || TopbaseLocale.locale() === 'zh-CN') return text;
  const retry = '；重试将使用已确认的回执恢复登记';
  if (text.endsWith(retry)) {
    return TopbaseLocale.t('{error}；重试将使用已确认的回执恢复登记', {error:warehouseErrorText(text.slice(0, -retry.length))});
  }
  const messages = [
    [/^恢复运行 (\S+) 的登记失败: ([\s\S]+)$/, '恢复运行 {run} 的登记失败: {error}', ['run','error']],
    [/^运行 (\S+) 的写入尚未核对，请管理员核对 (\S+) 的数据、水位与运行记录后恢复计划。([\s\S]*)$/, '运行 {run} 的写入尚未核对，请管理员核对 {target} 的数据、水位与运行记录后恢复计划。{error}', ['run','target','error']],
    [/^运行 (\S+) 需核对 (\S+)，已暂停重写目标表: ([\s\S]+)$/, '运行 {run} 需核对 {target}，已暂停重写目标表: {error}', ['run','target','error']],
    [/^写入回执未能登记，请管理员核对运行 (\S+) 和目标 (\S+): ([\s\S]+)$/, '写入回执未能登记，请管理员核对运行 {run} 和目标 {target}: {error}', ['run','target','error']],
    [/^目标库已提交 (\d+) 行，水位 ("(?:[^"\\]|\\.)*")，应用库登记未确认: ([\s\S]+)$/, '目标库已提交 {count} 行，水位 {watermark}，应用库登记未确认: {error}', ['count','watermark','error']],
    [/^运行 (\S+) 的写入屏障与历史记录不一致，请核对数据$/, '运行 {run} 的写入屏障与历史记录不一致，请核对数据', ['run']],
    [/^(检查上次写入状态失败|登记目标库写入意图失败，尚未写入目标表|读取目标表水位失败|登记失败运行状态|目标库提交结果未确认): ([\s\S]+)$/, '{reason}: {error}', ['reason','error']]
  ];
  for (const [pattern, key, names] of messages) {
    const match = pattern.exec(text);
    if (!match) continue;
    const args = {};
    names.forEach((name, index) => {
      const value = match[index+1];
      args[name] = name === 'error' ? warehouseErrorText(value) : name === 'reason' ? TopbaseLocale.t(value) : value;
    });
    return TopbaseLocale.t(key, args);
  }
  switch (text) {
    case '已有沉淀计划或数据表使用此目标，请更换保存表名':
    case '目标库提交结果未确认':
    case '非空增量目标表的水位为空，请检查变化字段':
    case 'PostgreSQL 的目标 schema 和表名最长各为 63 字节':
    case '沉淀计划缺少目标数据库连接':
    case '沉淀运行记录与目标表登记服务未配置':
      return TopbaseLocale.t(text);
    default:
      return text;
  }
}
