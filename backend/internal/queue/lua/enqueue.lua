-- Implements DESIGN-004 JobQueueManager idempotent, cluster-safe enqueue.
local existing = redis.call('get', KEYS[1])
if existing then
  return existing
end

local entry = redis.pcall('xadd', KEYS[2], '*', 'job_id', ARGV[1], 'enqueued_at', ARGV[2])
if type(entry) == 'table' and entry.err then
  return redis.error_reply(entry.err)
end

redis.call('set', KEYS[1], entry, 'px', ARGV[3])
return entry
