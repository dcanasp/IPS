using IPS;
using Microsoft.Extensions.Caching.Memory;

public class EnforcementMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IMemoryCache _cache;

    public EnforcementMiddleware(RequestDelegate next, IMemoryCache cache)
    {
        _next = next;
        _cache = cache;
    }

    public async Task Invoke(HttpContext context)
    {
        var ip = context.Connection.RemoteIpAddress?.ToString() ?? "unknown";
        var banKey = $"ban:{ip}";

        if (_cache.TryGetValue<BanInfo>(banKey, out var banInfo))
        {
            context.Response.StatusCode = 403;
            context.Response.ContentType = "application/json";

            var response = new
            {
                blocked = true,
                reason = banInfo.Reason,
                score = banInfo.Score,
                bannedAt = banInfo.BannedAt,
                expiresAt = banInfo.ExpiresAt
            };

            await context.Response.WriteAsJsonAsync(response);
            return;
        }

        await _next(context);
    }
}
