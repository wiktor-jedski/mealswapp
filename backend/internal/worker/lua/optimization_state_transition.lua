-- Implements DESIGN-004 JobStatusTracker atomic monotonic publication.
local currentPayload = redis.call('get', KEYS[1])
if not currentPayload then
  if redis.call('exists', KEYS[2]) == 1 then
    return -1
  end
  if ARGV[2] ~= 'save' then
    return -1
  end
  redis.call('set', KEYS[1], ARGV[1], 'px', ARGV[3])
  return 1
end

local current = cjson.decode(currentPayload)
local status = current.status
local operation = ARGV[2]
local allowed = false
if operation == 'save' then
  allowed = status == 'queued'
elseif operation == 'processing' then
  allowed = status == 'queued' or status == 'processing'
elseif operation == 'completed' or operation == 'failed' then
  allowed = status == 'processing'
end
if not allowed then
  return 0
end

redis.call('set', KEYS[1], ARGV[1], 'px', ARGV[3])
if operation == 'completed' or operation == 'failed' then
  redis.call('set', KEYS[2], current.userId, 'px', ARGV[5])
end
return 1
