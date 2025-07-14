using Microsoft.AspNetCore.HttpOverrides;

var builder = WebApplication.CreateBuilder(args);

// Add MemoryCache
builder.Services.AddMemoryCache();

// Add YARP
builder.Services.AddReverseProxy()
    .LoadFromMemory(
        new[]
        {
            new Yarp.ReverseProxy.Configuration.RouteConfig
            {
                RouteId = "default",
                ClusterId = "victim-cluster",
                Match = new() { Path = "{**catch-all}" }
            }
        },
        new[]
        {
            new Yarp.ReverseProxy.Configuration.ClusterConfig
            {
                ClusterId = "victim-cluster",
                Destinations = new Dictionary<string, Yarp.ReverseProxy.Configuration.DestinationConfig>
                {
                    { "dest1", new() { Address = "http://victim:8080/" } }
                }
            }
        });

var app = builder.Build();

// var cache = app.Services.GetRequiredService<IMemoryCache>();

// Trust real client IP from Docker network
app.UseForwardedHeaders(new ForwardedHeadersOptions
{
    ForwardedHeaders = ForwardedHeaders.XForwardedFor | ForwardedHeaders.XForwardedProto
});
app.Use(async (context, next) =>
{
    var ip = context.Connection.RemoteIpAddress?.ToString() ?? "unknown";
    Console.WriteLine(ip);
    await next();
});

app.UseMiddleware<EnforcementMiddlewaxre>();
app.UseMiddleware<TrafficLoggingMiddleware>();
app.UseMiddleware<ScoringMiddleware>();

// Ban middleware (basic version)
//app.Use(async (context, next) =>
//{
//    var ip = context.Connection.RemoteIpAddress?.ToString() ?? "unknown";

//    if (cache.TryGetValue($"ban:{ip}", out _))
//    {
//        context.Response.StatusCode = 403;
//        await context.Response.WriteAsync("Your IP has been banned.");
//        return;
//    }

//    await next();
//});

// Forward everything else
app.MapReverseProxy();

app.Run();
