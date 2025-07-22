using Microsoft.Extensions.Caching.Memory;

public class TrafficLoggingMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IMemoryCache _cache;

    public TrafficLoggingMiddleware(RequestDelegate next, IMemoryCache cache)
    {
        _next = next;
        _cache = cache;
    }

    public async Task Invoke(HttpContext context)
    {
        var ip = context.Connection.RemoteIpAddress?.ToString() ?? "unknown";
        var timestamp = DateTime.UtcNow;

        var record = new RequestRecord
        {
            Timestamp = timestamp,
            Method = context.Request.Method,
            Path = context.Request.Path.ToString(),
            Query = context.Request.QueryString.ToString(),
            Headers = context.Request.Headers.ToDictionary(h => h.Key, h => h.Value.ToString()),
            ContentLength = context.Request.ContentLength ?? 0,
            ResponseCode = context.Response.StatusCode
        };

        var key = $"log:{ip}";

        // Add or update log entry for this IP
        _cache.TryGetValue<List<RequestRecord>>(key, out var logs);
        logs ??= new List<RequestRecord>();

        logs.Add(record);

        // Keep only the last 10 minutes of logs
        var cutoff = timestamp.AddMinutes(-1);//HERE
        logs = logs.Where(r => r.Timestamp >= cutoff).ToList();

        _cache.Set(key, logs, new MemoryCacheEntryOptions
        {
            SlidingExpiration = TimeSpan.FromMinutes(1)//HERE
        });

        await _next(context);
    }
}

public class RequestRecord
{
    public DateTime Timestamp { get; set; }
    public string Method { get; set; } = string.Empty;
    public string Path { get; set; } = string.Empty;
    public string Query { get; set; } = string.Empty;
    public Dictionary<string, string> Headers { get; set; } = new();
    public long ContentLength { get; set; }
    public int ResponseCode { get; set; }
}
