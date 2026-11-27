// Used by the workspace and the administrative sharing overview.
function dashboardShareStatus(board) {
  if (!board?.public_uuid) return 'private';
  return Date.parse(board.public_expires_at) <= Date.now() ? 'expired' : 'published';
}
