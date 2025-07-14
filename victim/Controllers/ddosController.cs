using Microsoft.AspNetCore.Mvc;
using victim.Persistence;
using victim.Utils;
namespace victim.Controllers
{
    [ApiController]
    [Route("[controller]")]
    public class ddosController : ControllerBase
    {
        private static readonly string[] Summaries = new[]
        {
            "Freezing", "Bracing", "Chilly", "Cool", "Mild", "Warm", "Balmy", "Hot", "Sweltering", "Scorching"
        };

        private readonly ILogger<ddosController> _logger;
        private readonly AuthUtils _passwordUtils;
        private readonly Users _users;

        public ddosController(ILogger<ddosController> logger, AuthUtils passwordUtils, Users users)
        {
            _logger = logger;
            _passwordUtils = passwordUtils;
            _users = users;
        }

        [HttpGet(Name = "GetWeatherForecast")]
        public IEnumerable<WeatherForecast> Get()
        {
            return Enumerable.Range(1, 5).Select(index => new WeatherForecast
            {
                Date = DateOnly.FromDateTime(DateTime.Now.AddDays(index)),
                TemperatureC = Random.Shared.Next(-20, 55),
                Summary = Summaries[Random.Shared.Next(Summaries.Length)]
            })
            .ToArray();
        }

        [HttpPost("register")]
        public IActionResult Register([FromBody] Payload payload)
        {
            if (_users.GetUser(payload.Name) != null)
            {
                return BadRequest("User already exists");
            }

            var hashedPassword = _passwordUtils.HashPassword(payload.password);
            _users.AddUser(new User
            {
                Username = payload.Name,
                hashedPassword = hashedPassword,
                Email = payload.Email,
            });
            var token = _passwordUtils.GenerateJwtToken(payload.Name);

            var response = new JwtResponse
            {
                Token = token,
                Expiration = DateTime.UtcNow.AddMinutes(10)
            };

            return Ok(response);
        }



        [HttpPost("login")]
        public IActionResult Login([FromBody] Payload payload)
        {
            var user = _users.GetUser(payload.Name);
            if (user == null || !_passwordUtils.VerifyPassword(payload.password, user.hashedPassword))
            {
                return Unauthorized("Invalid username or password");
            }

            var token = _passwordUtils.GenerateJwtToken(payload.Name,user.Role);

            var response = new JwtResponse
            {
                Token = token,
                Expiration = DateTime.UtcNow.AddMinutes(10)
            };

            return Ok(response);
        }
        [HttpGet("users")]
        public IActionResult GetUsers()
        {
            var users = _users.GetAllUsers();
            return Ok(users);
        }
    }
}
