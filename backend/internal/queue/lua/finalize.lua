-- Implements DESIGN-004 JobQueueManager atomic terminal finalization.
local existing = redis.call('get', KEYS[1])
if existing and existing ~= ARGV[1] then
  return -2
end

local acknowledged = redis.call('xack', KEYS[2], ARGV[3], ARGV[4])
if acknowledged == 0 and not existing then
  return -1
end

if not existing then
  redis.call('set', KEYS[1], ARGV[1], 'px', ARGV[2])
end
redis.call('xdel', KEYS[2], ARGV[4])
return acknowledged
