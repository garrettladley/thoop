-- Dual-limit rate limiter for WHOOP API
-- Checks FOUR limits atomically: per-user minute/day + global minute/day
-- Increments all counters only if ALL limits pass
--
-- KEYS[1]: per-user minute key (e.g., "whoop:ratelimit:user:abc123:minute")
-- KEYS[2]: per-user day key (e.g., "whoop:ratelimit:user:abc123:day")
-- KEYS[3]: global minute key (e.g., "whoop:ratelimit:global:minute")
-- KEYS[4]: global day key (e.g., "whoop:ratelimit:global:day")
--
-- ARGV[1]: per_user_minute_limit (e.g., 20)
-- ARGV[2]: per_user_day_limit (e.g., 2000)
-- ARGV[3]: global_minute_limit (e.g., 95)
-- ARGV[4]: global_day_limit (e.g., 9950)
-- ARGV[5]: minute_window_ms (60000 = 1 minute)
-- ARGV[6]: day_window_ms (86400000 = 24 hours)
-- ARGV[7]: ttl_seconds (e.g., 90000 for day + margin)
--
-- Returns:
-- {1, minute_remaining, day_remaining} if allowed
-- {0, "reason"} if blocked, where reason is one of:
--   "per-user-minute", "per-user-day", "global-minute", "global-day"

local user_min_key = KEYS[1]
local user_day_key = KEYS[2]
local global_min_key = KEYS[3]
local global_day_key = KEYS[4]

local user_min_limit = tonumber(ARGV[1])
local user_day_limit = tonumber(ARGV[2])
local global_min_limit = tonumber(ARGV[3])
local global_day_limit = tonumber(ARGV[4])
local min_window_ms = tonumber(ARGV[5])
local day_window_ms = tonumber(ARGV[6])
local ttl = tonumber(ARGV[7])

local time_result = redis.call('TIME')
local now = tonumber(time_result[1]) * 1000 + math.floor(tonumber(time_result[2]) / 1000)

local min_window_start = now - min_window_ms
local day_window_start = now - day_window_ms

-- clean up expired entries
redis.call('ZREMRANGEBYSCORE', user_min_key, '-inf', min_window_start)
redis.call('ZREMRANGEBYSCORE', user_day_key, '-inf', day_window_start)
redis.call('ZREMRANGEBYSCORE', global_min_key, '-inf', min_window_start)
redis.call('ZREMRANGEBYSCORE', global_day_key, '-inf', day_window_start)

-- count current requests
local user_min_count = redis.call('ZCARD', user_min_key)
local user_day_count = redis.call('ZCARD', user_day_key)
local global_min_count = redis.call('ZCARD', global_min_key)
local global_day_count = redis.call('ZCARD', global_day_key)

-- check all four limits
if user_min_count >= user_min_limit then
    return { 0, "per-user-minute" }
end

if user_day_count >= user_day_limit then
    return { 0, "per-user-day" }
end

if global_min_count >= global_min_limit then
    return { 0, "global-minute" }
end

if global_day_count >= global_day_limit then
    return { 0, "global-day" }
end

-- all limits passed - increment all counters
local member = tostring(now) .. ':' .. tostring(math.random(1000000))

redis.call('ZADD', user_min_key, now, member)
redis.call('ZADD', user_day_key, now, member .. ':day')
redis.call('ZADD', global_min_key, now, member .. ':global')
redis.call('ZADD', global_day_key, now, member .. ':global:day')

-- set expiration
redis.call('EXPIRE', user_min_key, ttl)
redis.call('EXPIRE', user_day_key, ttl)
redis.call('EXPIRE', global_min_key, ttl)
redis.call('EXPIRE', global_day_key, ttl)

-- calculate remaining
local min_remaining = global_min_limit - global_min_count - 1
local day_remaining = global_day_limit - global_day_count - 1

return { 1, min_remaining, day_remaining }
