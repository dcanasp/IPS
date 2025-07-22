
public class RiskAssessor
{
    // --- Define your weights and normalization maximums ---
    private const double W_REQUEST_RATE = 0.4;
    private const double W_ERROR_RATE = 0.3;
    private const double W_HEADER_CHANGE_RATE = 0.1;
    private const double W_STDDEV_PAYLOAD = 0.1;
    private const double W_UNIQUE_PATHS = 0.1;

    private const double MAX_REQUEST_RATE = 300.0;
    private const double MAX_ERROR_RATE = 0.75;
    private const double MAX_HEADER_CHANGE_RATE = 5.0;
    private const double MAX_STDDEV_PAYLOAD = 15000.0;
    private const double MAX_UNIQUE_PATHS = 9.0;
    public double BAN_THRESHOLD { get; } = 0.6;

    public double CalculateRiskScore(double requestRate, double errorRate, double headerChangeRate, double stddevPayload, double uniquePaths)
    {
        // 1. Normalize each metric
        var nRequestRate = Math.Min(1.0, requestRate / MAX_REQUEST_RATE);
        var nErrorRate = Math.Min(1.0, errorRate / MAX_ERROR_RATE);
        var nHeaderChangeRate = Math.Min(1.0, headerChangeRate / MAX_HEADER_CHANGE_RATE);
        var nStddevPayload = Math.Min(1.0, Math.Max(1.0, stddevPayload / MAX_STDDEV_PAYLOAD));//esto siempre dara 1
        var nUniquePaths = 1 - Math.Min(1.0, uniquePaths / MAX_UNIQUE_PATHS);

        // 2. Calculate the weighted sum
        double riskScore =
            (W_REQUEST_RATE * nRequestRate) +
            (W_ERROR_RATE * nErrorRate) +
            (W_HEADER_CHANGE_RATE * nHeaderChangeRate) +
            (W_STDDEV_PAYLOAD * nStddevPayload) +
            (W_UNIQUE_PATHS * nUniquePaths);

        return riskScore;
    }
}
