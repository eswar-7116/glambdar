local key = KEYS[1]                      -- function's ratelimit key
local capacity = tonumber(ARGV[1])       -- capacity (max tokens)
local refill_rate = tonumber(ARGV[2])    -- refill_rate (tokens per second)
local now = tonumber(ARGV[3])            -- current time in milliseconds
local requested = tonumber(ARGV[4])      -- requested tokens

local fill_time = capacity / refill_rate -- seconds to fill from empty

local ratelimit_info = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(ratelimit_info[1])
local last_refill = tonumber(ratelimit_info[2])

-- Initialize if not exists
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Refill based on elapsed time
local time_passed = math.max(0, now - last_refill)
local refill_amount = math.floor(time_passed * (refill_rate / 1000))

if refill_amount > 0 then
    tokens = math.min(capacity, tokens + refill_amount)
    last_refill = now
end

-- Check if we have enough tokens
local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
end

-- Save state and extend TTL to 2x the time it takes to fill the bucket
redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
redis.call("PEXPIRE", key, math.ceil(fill_time * 1000 * 2))

return allowed
