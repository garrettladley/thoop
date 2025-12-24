-- Sliding window rate limiter
-- KEYS[1]: rate limit key (e.g., "ratelimit:192.168.1.1")
-- ARGV[1]: window_ms - sliding window size in milliseconds
-- ARGV[2]: limit - max requests allowed in window
-- ARGV[3]: ttl - key expiration in seconds

local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

local time_result = redis.call('TIME')
local now = tonumber(time_result[1]) * 1000 + math.floor(tonumber(time_result[2]) / 1000)
local window_start = now - window_ms

redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)

if count < limit then
    local member = tostring(now) .. ':' .. tostring(math.random(1000000))
    redis.call('ZADD', key, now, member)
    redis.call('EXPIRE', key, ttl)
    return 1
else
    return 0
end
