namespace victim.Utils
{
    using Microsoft.AspNetCore.Cryptography.KeyDerivation;
    using System.Security.Cryptography;
    using System.IdentityModel.Tokens.Jwt;
    using System.Security.Claims;
    using Microsoft.IdentityModel.Tokens;
    using System.Text;
    using victim.Persistence;

    public class AuthUtils
    {
        public string HashPassword(string password)
        {

            string hashed = Convert.ToBase64String(KeyDerivation.Pbkdf2(
                password: password!,
                salt: Array.Empty<byte>(),
                prf: KeyDerivationPrf.HMACSHA256,
                iterationCount: 100000,
                numBytesRequested: 256 / 8));
            return hashed;
        }

        public bool VerifyPassword(string password, string hashedPassword)
        {
            return HashPassword(password) == hashedPassword;
        }
        public string GenerateJwtToken(string username, UserRole role = UserRole.User)
        {
            var key = Encoding.UTF8.GetBytes("FakePasswordbutIthasToBeLongEnough123!");
            var creds = new SigningCredentials(new SymmetricSecurityKey(key), SecurityAlgorithms.HmacSha256);

            var token = new JwtSecurityToken(
                issuer: "burned-app",
                audience: null,
                claims: new[] { new Claim(ClaimTypes.Name, username), new Claim(ClaimTypes.Role, role.ToString()) },
                expires: DateTime.UtcNow.AddMinutes(10),
                signingCredentials: creds
            );

            return new JwtSecurityTokenHandler().WriteToken(token);
        }

    }
}