using IPS;
using Microsoft.Extensions.Caching.Memory;

public class ScoringMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IMemoryCache _cache;
    private readonly RiskAssessor _riskAssessor;

    public ScoringMiddleware(RequestDelegate next, IMemoryCache cache, RiskAssessor riskAssessor)
    {
        _next = next;
        _cache = cache;
        _riskAssessor = riskAssessor;
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
        var requestRate = logs.Count / 1.0; //HERE
        var uniquePaths = logs.Select(r => r.Path).Distinct().Count();
        var headerChangeRate = logs
            .Select(r => r.Headers.GetValueOrDefault("Authorization", "") +
                          r.Headers.GetValueOrDefault("User-Agent", ""))
            .Distinct().Count();
        var avgContentLength = logs.Select(r => r.ContentLength).DefaultIfEmpty(0).Average();
        var stddevPayload = Math.Sqrt(logs
            .Select(r => Math.Pow(r.ContentLength - avgContentLength, 2))
            .DefaultIfEmpty(0).Average());

        var invalidPaths = logs.Count(r =>
            r.Path.StartsWith("/admin") || r.Path.Contains("..") || r.Path.StartsWith("/undefined"));
        var FailedResponseCodes = logs.Select(r => r.ResponseCode)
            .Where(code => code is >= 400 and < 600).Count();
        var errorRate = (double)(invalidPaths + FailedResponseCodes) / logs.Count;

        var riskScore = _riskAssessor.CalculateRiskScore(requestRate, errorRate, headerChangeRate, stddevPayload, uniquePaths);

        //// ---- Scoring logic ----
        //double score = 0;
        //score += requestRate * 1.0;
        //score += uniquePaths > 10 ? 2 : 0;
        //score += headerChangeRate > 3 ? 2 : 0;
        //score += stddevPayload < 10 ? 2 : 0;
        //score += errorRate > 0.2 ? 2 : 0;

        if (riskScore > _riskAssessor.BAN_THRESHOLD)
        {
            var banInfo = new BanInfo
            {
                Reason = "High score from abnormal behavior",
                Score = riskScore,
                BannedAt = now,
                ExpiresAt = now.AddMinutes(10)
            };
            _cache.Set($"ban:{ip}", banInfo, TimeSpan.FromMinutes(1));//HERE
            Console.WriteLine($"[!] Blocking IP: {ip} | Score: {riskScore:F2}");
            context.Response.StatusCode = 403;
            await context.Response.WriteAsync("Blocked by IPS.");
            return;
        }

        await _next(context);
    }
}
