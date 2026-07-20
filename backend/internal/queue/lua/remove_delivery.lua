-- Implements DESIGN-004 JobQueueManager atomic duplicate/malformed cleanup.
local acknowledged = redis.call('xack', KEYS[1], ARGV[1], ARGV[2])
local deleted = redis.call('xdel', KEYS[1], ARGV[2])
return {acknowledged, deleted}
