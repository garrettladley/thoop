-- Global-only rate limiter for WHOOP API
-- Checks only global limits (not per-user) for operations like token validation
--
-- KEYS[1]: global minute key (e.g., "whoop:ratelimit:global:minute")
-- KEYS[2]: global day key (e.g., "whoop:ratelimit:global:day")
--
-- ARGV[1]: global_minute_limit (e.g., 95)
-- ARGV[2]: global_day_limit (e.g., 9950)
-- ARGV[3]: minute_window_ms (60000 = 1 minute)
-- ARGV[4]: day_window_ms (86400000 = 24 hours)
-- ARGV[5]: ttl_seconds (e.g., 90000 for day + margin)
--
-- Returns:
-- {1, minute_remaining, day_remaining} if allowed
-- {0, "reason"} if blocked, where reason is one of:
--   "global-minute", "global-day"

local global_min_key = KEYS[1]
local global_day_key = KEYS[2]

local global_min_limit = tonumber(ARGV[1])
local global_day_limit = tonumber(ARGV[2])
local min_window_ms = tonumber(ARGV[3])
local day_window_ms = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local time_result = redis.call('TIME')
local now = tonumber(time_result[1]) * 1000 + math.floor(tonumber(time_result[2]) / 1000)

local min_window_start = now - min_window_ms
local day_window_start = now - day_window_ms

-- clean up expired entries
redis.call('ZREMRANGEBYSCORE', global_min_key, '-inf', min_window_start)
redis.call('ZREMRANGEBYSCORE', global_day_key, '-inf', day_window_start)

-- count current requests
local global_min_count = redis.call('ZCARD', global_min_key)
local global_day_count = redis.call('ZCARD', global_day_key)

-- check global limits
if global_min_count >= global_min_limit then
    return { 0, "global-minute" }
end

if global_day_count >= global_day_limit then
    return { 0, "global-day" }
end

-- limits passed - increment global counters only
local member = tostring(now) .. ':' .. tostring(math.random(1000000))

redis.call('ZADD', global_min_key, now, member .. ':global')
redis.call('ZADD', global_day_key, now, member .. ':global:day')

-- set expiration
redis.call('EXPIRE', global_min_key, ttl)
redis.call('EXPIRE', global_day_key, ttl)

-- calculate remaining
local min_remaining = global_min_limit - global_min_count - 1
local day_remaining = global_day_limit - global_day_count - 1

return { 1, min_remaining, day_remaining }
