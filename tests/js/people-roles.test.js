const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');

test('people list labels system and delegated administrators using the loaded group memberships', () => {
  const html = fs.readFileSync('internal/platform/httpapi/web/admin/people/index.html', 'utf8');
  const render = html.slice(html.indexOf('function groupPermission('), html.indexOf('async function load()'));
  const users = ['system', 'delegated', 'regular'].map(id=>({ id, name:id, email:id+'@example.invalid' }));
  const result = vm.runInNewContext(render+';userTable(users)', {
    users, groups:[
      {id:'grp_admins', member_ids:['system']}, {id:'delegates', member_ids:['delegated']},
      {id:'everyone', member_ids:users.map(u=>u.id)}
    ],
    permissionGraph:{data_graph:{delegates:{admin:'admin'}}},
    userGroupsHTML:()=>'', esc:String, initials:String
  });
  assert.match(result, /system<\/strong><small>管理员/);
  assert.match(result, /delegated<\/strong><small>管理员/);
  assert.match(result, /regular<\/strong><small>成员/);
});
