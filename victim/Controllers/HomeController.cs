using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Authorization;

namespace victim.Controllers
{
    [ApiController]
    public class HomeController : ControllerBase
    {
        [HttpGet("/")]
        public IActionResult Index()
        {
            return Ok("Welcome to the public home page.");
        }

        [Authorize]
        [HttpGet("/profile")]
        public IActionResult Profile()
        {
            var username = User.Identity?.Name;
            return Ok(new { Message = $"Welcome to your profile, {username}!" });
        }

        [HttpPost("/logout")]
        public IActionResult Logout()
        {
            // Invalidate token logic would go here (not implemented for simplicity)
            return Ok(new { Message = "Logged out (token should be deleted on client)." });
        }
    }
}
