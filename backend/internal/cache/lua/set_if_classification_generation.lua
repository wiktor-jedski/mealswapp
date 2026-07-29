-- Implements DESIGN-011 RedisCache guarded cache-miss persistence.
local current = redis.call("GET", KEYS[1])
if (not current and ARGV[1] == "0") or current == ARGV[1] then
  redis.call("SET", KEYS[2], ARGV[2], "PX", ARGV[3])
  return 1
end
return 0
