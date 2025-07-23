using System.Text.Json;
using Serilog.Events;
namespace IPS 
{
    public class OpenTelemetryLikeJsonFormatter : Serilog.Formatting.ITextFormatter
    {
        public void Format(LogEvent logEvent, TextWriter output)
        {
            var resourceAttributes = new Dictionary<string, object>
        {
            {"service.name", "VictimService"},
            {"service.instance.id", Environment.MachineName} // Or a unique ID for the container instance
        };

            // Add any additional properties from Serilog's logEvent
            var attributes = new Dictionary<string, object>();
            foreach (var prop in logEvent.Properties)
            {
                // Convert Serilog property values to their literal value if possible, or string
                if (prop.Value is ScalarValue scalar)
                {
                    attributes[prop.Key] = scalar.Value;
                }
                else
                {
                    attributes[prop.Key] = prop.Value.ToString();
                }
            }

            var otelLogEntry = new
            {
                timestamp = logEvent.Timestamp.ToUniversalTime().ToString("o"), // ISO 8601 format
                traceId = logEvent.Properties.ContainsKey("TraceId") ? logEvent.Properties["TraceId"].ToString().Replace("\"", "") : Guid.NewGuid().ToString("N"), // Remove quotes if present, default to new GUID
                spanId = logEvent.Properties.ContainsKey("SpanId") ? logEvent.Properties["SpanId"].ToString().Replace("\"", "") : Guid.NewGuid().ToString("N"), // Remove quotes if present, default to new GUID
                severityText = logEvent.Level.ToString(),
                body = logEvent.RenderMessage(),
                resource = resourceAttributes,
                attributes = attributes
            };

            output.WriteLine(JsonSerializer.Serialize(otelLogEntry, new JsonSerializerOptions { WriteIndented = false }));
        }
    }
}
