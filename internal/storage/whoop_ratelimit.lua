-- Dual-limit rate limiter for WHOOP API with dynamic per-user limits
-- Checks FOUR limits atomically: per-user minute/day + global minute/day
-- Tracks active users to dynamically calculate per-user minute limits
-- Increments all counters only if ALL limits pass
--
-- KEYS[1]: per-user minute key (e.g., "whoop:ratelimit:user:abc123:minute")
-- KEYS[2]: per-user day key (e.g., "whoop:ratelimit:user:abc123:day")
-- KEYS[3]: global minute key (e.g., "whoop:ratelimit:global:minute")
-- KEYS[4]: global day key (e.g., "whoop:ratelimit:global:day")
-- KEYS[5]: active users key (e.g., "whoop:ratelimit:active_users")
--
-- ARGV[1]: per_user_day_limit (e.g., 1000)
-- ARGV[2]: global_minute_limit (e.g., 95)
-- ARGV[3]: global_day_limit (e.g., 9950)
-- ARGV[4]: minute_window_ms (60000 = 1 minute)
-- ARGV[5]: day_window_ms (86400000 = 24 hours)
-- ARGV[6]: ttl_seconds (e.g., 90000 for day + margin)
-- ARGV[7]: reserve_buffer (e.g., 5)
-- ARGV[8]: min_per_user_limit (e.g., 5)
-- ARGV[9]: active_window_ms (e.g., 60000)
-- ARGV[10]: user_id (string for active set member)
--
-- Returns:
-- {1, minute_remaining, day_remaining, dynamic_limit, active_count} if allowed
-- {0, "reason", dynamic_limit, active_count} if blocked, where reason is one of:
--   "per-user-minute", "per-user-day", "global-minute", "global-day"

local user_min_key = KEYS[1]
local user_day_key = KEYS[2]
local global_min_key = KEYS[3]
local global_day_key = KEYS[4]
local active_users_key = KEYS[5]

local user_day_limit = tonumber(ARGV[1])
local global_min_limit = tonumber(ARGV[2])
local global_day_limit = tonumber(ARGV[3])
local min_window_ms = tonumber(ARGV[4])
local day_window_ms = tonumber(ARGV[5])
local ttl = tonumber(ARGV[6])
local reserve_buffer = tonumber(ARGV[7])
local min_per_user_limit = tonumber(ARGV[8])
local active_window_ms = tonumber(ARGV[9])
local user_id = ARGV[10]

local time_result = redis.call('TIME')
local now = tonumber(time_result[1]) * 1000 + math.floor(tonumber(time_result[2]) / 1000)

local min_window_start = now - min_window_ms
local day_window_start = now - day_window_ms
local active_window_start = now - active_window_ms

-- clean up expired entries
redis.call('ZREMRANGEBYSCORE', user_min_key, '-inf', min_window_start)
redis.call('ZREMRANGEBYSCORE', user_day_key, '-inf', day_window_start)
redis.call('ZREMRANGEBYSCORE', global_min_key, '-inf', min_window_start)
redis.call('ZREMRANGEBYSCORE', global_day_key, '-inf', day_window_start)

-- clean expired active users
redis.call('ZREMRANGEBYSCORE', active_users_key, '-inf', active_window_start)

-- update current user's activity timestamp
redis.call('ZADD', active_users_key, now, user_id)
redis.call('EXPIRE', active_users_key, 120)

-- count active users and calculate dynamic limit
local active_count = redis.call('ZCARD', active_users_key)
local available = global_min_limit - reserve_buffer
local dynamic_limit = math.floor(available / active_count)
dynamic_limit = math.max(dynamic_limit, min_per_user_limit)

-- count current requests
local user_min_count = redis.call('ZCARD', user_min_key)
local user_day_count = redis.call('ZCARD', user_day_key)
local global_min_count = redis.call('ZCARD', global_min_key)
local global_day_count = redis.call('ZCARD', global_day_key)

-- check all four limits (using dynamic_limit for per-user minute)
if user_min_count >= dynamic_limit then
    return { 0, "per-user-minute", dynamic_limit, active_count }
end

if user_day_count >= user_day_limit then
    return { 0, "per-user-day", dynamic_limit, active_count }
end

if global_min_count >= global_min_limit then
    return { 0, "global-minute", dynamic_limit, active_count }
end

if global_day_count >= global_day_limit then
    return { 0, "global-day", dynamic_limit, active_count }
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

return { 1, min_remaining, day_remaining, dynamic_limit, active_count }
