using System.Text;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.IdentityModel.Tokens;
using Microsoft.OpenApi.Models;
using Serilog;
using Serilog.Events;
using victim;
using victim.Persistence;
var builder = WebApplication.CreateBuilder(args);


// Add services to the container.
builder.Services.AddControllers();
UserList.Init();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(options =>
            {
                options.SwaggerDoc("v1", new OpenApiInfo
                {
                    Title = "Vinnare API",
                    Version = "v1"
                });

                var securityScheme = new OpenApiSecurityScheme
                {
                    Name = "Authorization",
                    Description = "Enter the JWT token in this format: Bearer {your_token}",
                    In = ParameterLocation.Header,
                    Type = SecuritySchemeType.Http,
                    Scheme = "bearer",
                    BearerFormat = "JWT",
                    Reference = new OpenApiReference
                    {
                        Type = ReferenceType.SecurityScheme,
                        Id = "Bearer"
                    }
                };

                var securityRequirement = new OpenApiSecurityRequirement
                {
                    {
                        securityScheme,
                        new string[] { }
                    }
                };

                options.AddSecurityDefinition("Bearer", securityScheme);
                options.AddSecurityRequirement(securityRequirement);
            });

Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Debug() // Set your desired minimum level
    .Enrich.FromLogContext() // Allows adding properties dynamically
                             // You can add properties like TraceId and SpanId via log context if you integrate with a tracing system later
    .WriteTo.File(
        formatter: new OpenTelemetryLikeJsonFormatter(),
        path: "/app/logs/victim-service.json", // Path inside the Docker container
        rollingInterval: RollingInterval.Day, // Or None, depending on expected log volume for 30 min
        restrictedToMinimumLevel: LogEventLevel.Information, // Only log info and above to file
        buffered: false // For low resource, potentially better to write directly
    )
    .WriteTo.Console() // Keep console output for debugging during development
    .CreateLogger();

builder.Logging.ClearProviders(); // Clear default .NET Core loggers
builder.Logging.AddSerilog();

builder.Services.AddScoped<victim.Utils.AuthUtils>();
builder.Services.AddScoped<Users>();
builder.Services.AddHealthChecks();


var jwtKey = "FakePasswordbutIthasToBeLongEnough123!";
var jwtIssuer = "burned-app";

// Register JWT authentication
builder.Services.AddAuthentication(options =>
{
    options.DefaultAuthenticateScheme = JwtBearerDefaults.AuthenticationScheme;
    options.DefaultChallengeScheme = JwtBearerDefaults.AuthenticationScheme;
})
.AddJwtBearer(options =>
{
    options.RequireHttpsMetadata = false; // for local testing
    options.SaveToken = true;
    options.TokenValidationParameters = new TokenValidationParameters
    {
        ValidateIssuer = true,
        ValidateAudience = false,
        ValidateIssuerSigningKey = true,
        ValidIssuer = jwtIssuer,
        IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwtKey))
    };
});



var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();
app.MapHealthChecks("/health");

// Optional sanity check endpoint
app.MapGet("/sanity", () => Results.Ok(ProductStore.SumValues()));
app.Run();
