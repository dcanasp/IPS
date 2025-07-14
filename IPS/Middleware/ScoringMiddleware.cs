using IPS;
using Microsoft.Extensions.Caching.Memory;

public class ScoringMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IMemoryCache _cache;

    public ScoringMiddleware(RequestDelegate next, IMemoryCache cache)
    {
        _next = next;
        _cache = cache;
    }

    public async Task Invoke(HttpContext context)
    {
        var ip = context.Connection.RemoteIpAddress?.ToString() ?? "unknown";
        var now = DateTime.UtcNow;

        var key = $"log:{ip}";
        if (!_cache.TryGetValue<List<RequestRecord>>(key, out var logs) || logs.Count == 0)
        {
            await _next(context);
            return;
        }

        // Clean logs older than 10 minutes
        logs = logs.Where(r => r.Timestamp >= now.AddMinutes(-10)).ToList();
        _cache.Set(key, logs); // Update

        // ---- Feature extraction ----
        var requestRate = logs.Count / 1.0; // per minute
        var uniquePaths = logs.Select(r => r.Path).Distinct().Count();
        var headerChangeRate = logs.Select(r => r.Headers.GetValueOrDefault("User-Agent", "")).Distinct().Count();
        var avgContentLength = logs.Select(r => r.ContentLength).DefaultIfEmpty(0).Average();
        var stddevPayload = Math.Sqrt(logs
            .Select(r => Math.Pow(r.ContentLength - avgContentLength, 2))
            .DefaultIfEmpty(0).Average());

        var invalidPaths = logs.Count(r =>
            r.Path.StartsWith("/admin") || r.Path.Contains("..") || r.Path.StartsWith("/undefined"));

        var errorRate = (double)invalidPaths / logs.Count;

        // ---- Scoring logic ----
        double score = 0;
        score += requestRate * 1.0;
        score += uniquePaths > 10 ? 2 : 0;
        score += headerChangeRate > 3 ? 2 : 0;
        score += stddevPayload < 10 ? 2 : 0; // All payloads the same
        score += errorRate > 0.2 ? 2 : 0;

        if (score > 10)
        {
            var banInfo = new BanInfo
            {
                Reason = "High score from abnormal behavior",
                Score = score,
                BannedAt = now,
                ExpiresAt = now.AddMinutes(10)
            };
            _cache.Set($"ban:{ip}", banInfo, TimeSpan.FromMinutes(10));
            Console.WriteLine($"[!] Blocking IP: {ip} | Score: {score:F2}");
            context.Response.StatusCode = 403;
            await context.Response.WriteAsync("Blocked by IPS.");
            return;
        }

        await _next(context);
    }
}
