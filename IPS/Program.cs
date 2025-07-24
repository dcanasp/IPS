using IPS;
using Microsoft.AspNetCore.HttpOverrides;
using Serilog;
using Serilog.Events;
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddSingleton<RiskAssessor>();
builder.Services.AddMemoryCache();


builder.Services.AddCors(options =>
{
    options.AddPolicy("AllowAll", policy =>
    {
        policy
            .AllowAnyOrigin()
            .AllowAnyMethod()
            .AllowAnyHeader();
    });
});


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
                    //{ "dest1", new() { Address = "http://localhost:5282/" } }
                    { "dest1", new() { Address = "http://victim:8080/" } }

                }
            }
        });

Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Debug() // Set your desired minimum level
    .Enrich.FromLogContext() // Allows adding properties dynamically
                             // You can add properties like TraceId and SpanId via log context if you integrate with a tracing system later
    .WriteTo.File(
        formatter: new OpenTelemetryLikeJsonFormatter(),
        path: "/app/logs/ips-service.json", // Path inside the Docker container
        rollingInterval: RollingInterval.Day, // Or None, depending on expected log volume for 30 min
        restrictedToMinimumLevel: LogEventLevel.Information, // Only log info and above to file
        buffered: false // For low resource, potentially better to write directly
    )
    .WriteTo.Console() // Keep console output for debugging during development
    .CreateLogger();

builder.Logging.ClearProviders(); // Clear default .NET Core loggers
builder.Logging.AddSerilog();

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

app.UseMiddleware<EnforcementMiddleware>();
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
app.UseCors("AllowAll");
app.MapReverseProxy();

app.Run();
