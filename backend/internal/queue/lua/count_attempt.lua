-- Implements DESIGN-004 JobQueueManager atomic attempt accounting.
local current = redis.call('get', KEYS[1])
local attempt = 1
if current then
  attempt = tonumber(current)
  if not attempt then
    return redis.error_reply('invalid optimization attempt counter')
  end
  attempt = attempt + 1
end

redis.call('set', KEYS[1], tostring(attempt), 'px', ARGV[1])
return attempt
