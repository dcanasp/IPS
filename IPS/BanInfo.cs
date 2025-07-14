namespace IPS;
public class BanInfo
{
    public string Reason { get; set; } = "Unspecified";
    public double Score { get; set; }
    public DateTime BannedAt { get; set; }
    public DateTime ExpiresAt { get; set; }
}
